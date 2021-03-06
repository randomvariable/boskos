/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/logrusutil"
	"sigs.k8s.io/boskos/client"
	"sigs.k8s.io/boskos/common"
)

var (
	rTypes         common.CommaSeparatedStrings
	boskosURL      = flag.String("boskos-url", "http://boskos", "Boskos URL")
	username       = flag.String("username", "", "Username used to access the Boskos server")
	passwordFile   = flag.String("password-file", "", "The path to password file used to access the Boskos server")
	expiryDuration = flag.Duration("expire", 30*time.Minute, "The expiry time (in minutes) after which reaper will reset resources.")
	targetState    = flag.String("target-state", common.Dirty, "The state to move resources to when reaped.")
)

func init() {
	flag.Var(&rTypes, "resource-type", "comma-separated list of resources need to be reset")
}

func main() {
	logrusutil.ComponentInit()

	flag.Parse()
	boskos, err := client.NewClient("Reaper", *boskosURL, *username, *passwordFile)
	if err != nil {
		logrus.WithError(err).Fatal("unable to create a Boskos client")
	}
	logrus.Infof("Initialized boskos client!")

	if len(rTypes) == 0 {
		logrus.Fatal("--resource-type must not be empty!")
	}

	if targetState == nil {
		logrus.Fatal("--target-state must not be empty!")
	}

	for range time.Tick(time.Minute) {
		for _, r := range rTypes {
			sync(boskos, r)
		}
	}
}

func sync(c *client.Client, res string) {
	log := logrus.WithField("resource_type", res).WithField("target_state", *targetState)

	// kubetest busted
	log = log.WithField("source_state", common.Busy)
	if owners, err := c.Reset(res, common.Busy, *expiryDuration, *targetState); err != nil {
		log.WithError(err).Error("Reset failed")
	} else {
		logResponses(log, owners)
	}

	// janitor, mason busted
	log = log.WithField("source_state", common.Cleaning)
	if owners, err := c.Reset(res, common.Cleaning, *expiryDuration, *targetState); err != nil {
		log.WithError(err).Error("Reset failed")
	} else {
		logResponses(log, owners)
	}

	// mason busted
	log = log.WithField("source_state", common.Leased)
	if owners, err := c.Reset(res, common.Leased, *expiryDuration, *targetState); err != nil {
		log.WithError(err).Error("Reset failed")
	} else {
		logResponses(log, owners)
	}
}

func logResponses(parent *logrus.Entry, response map[string]string) {
	for name, previousOwner := range response {
		parent.WithField("resource_name", name).WithField("previous_owner", previousOwner).Info("Reset resource")
	}
}
