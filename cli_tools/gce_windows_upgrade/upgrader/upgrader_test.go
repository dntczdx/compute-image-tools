//  Copyright 2020 Google Inc. All Rights Reserved.
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package upgrader

import (
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/compute-image-tools/daisy"
)

type TestUpgrader struct {
	*Upgrader

	initFn              func() error
	printUpgradeGuideFn func() error
	validateParamsFn    func() error
	prepareFn           func() (*daisy.Workflow, error)
	upgradeFn           func() (*daisy.Workflow, error)
	retryUpgradeFn      func() (*daisy.Workflow, error)
	rebootFn            func() (*daisy.Workflow, error)
	cleanupFn           func() (*daisy.Workflow, error)
	rollbackFn          func() (*daisy.Workflow, error)
}

func (tu *TestUpgrader) getUpgrader() *Upgrader {
	return tu.Upgrader
}

func (tu *TestUpgrader) init() error {
	if tu.initFn == nil {
		return tu.Upgrader.init()
	}
	return tu.initFn()
}

func (tu *TestUpgrader) printUpgradeGuide() error {
	if tu.printUpgradeGuideFn == nil {
		return tu.Upgrader.printUpgradeGuide()
	}
	return tu.printUpgradeGuideFn()
}

func (tu *TestUpgrader) validateParams() error {
	if tu.validateParamsFn == nil {
		return tu.Upgrader.validateParams()
	}
	return tu.validateParamsFn()
}

func (tu *TestUpgrader) prepare() (*daisy.Workflow, error) {
	if tu.prepareFn == nil {
		return tu.Upgrader.prepare()
	}
	return tu.prepareFn()
}

func (tu *TestUpgrader) upgrade() (*daisy.Workflow, error) {
	if tu.upgradeFn == nil {
		return tu.Upgrader.upgrade()
	}
	return tu.upgradeFn()
}

func (tu *TestUpgrader) retryUpgrade() (*daisy.Workflow, error) {
	if tu.retryUpgradeFn == nil {
		return tu.Upgrader.retryUpgrade()
	}
	return tu.retryUpgradeFn()
}

func (tu *TestUpgrader) reboot() (*daisy.Workflow, error) {
	if tu.rebootFn == nil {
		return tu.Upgrader.reboot()
	}
	return tu.rebootFn()
}

func (tu *TestUpgrader) cleanup() (*daisy.Workflow, error) {
	if tu.cleanupFn == nil {
		return tu.Upgrader.cleanup()
	}
	return tu.cleanupFn()
}

func (tu *TestUpgrader) rollback() (*daisy.Workflow, error) {
	if tu.rollbackFn == nil {
		return tu.Upgrader.rollback()
	}
	return tu.rollbackFn()
}

func TestUpgraderRunFailedOnInit(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.initFn = nil
	tu.Oauth = "bad-oauth"

	_, err := Run(tu)
	if err == nil {
		t.Errorf("Expect error but none.")
	}
}

func TestUpgraderRunFailedOnValidateParams(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.validateParamsFn = func() error {
		return fmt.Errorf("failed")
	}

	_, err := Run(tu)
	if err == nil {
		t.Errorf("Expect error but none.")
	} else if err.Error() != "failed" {
		t.Errorf("Error not thrown from expected function.")
	}
}

func TestUpgraderRunFailedOnPrintUpgradeGuide(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.printUpgradeGuideFn = func() error {
		return fmt.Errorf("failed")
	}

	_, err := Run(tu)
	if err == nil {
		t.Errorf("Expect error but none.")
	} else if err.Error() != "failed" {
		t.Errorf("Error not thrown from expected function.")
	}
}

func TestUpgraderRunFailedOnPrepare(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.prepareFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("failed")
	}
	tu.cleanupFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}

	_, err := Run(tu)
	if err == nil {
		t.Errorf("Expect error but none.")
	} else if err.Error() != "failed" {
		t.Errorf("Error not thrown from expected function.")
	}
}

func TestUpgraderRunFailedOnUpgrade(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.upgradeFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("failed")
	}
	tu.cleanupFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}

	_, err := Run(tu)
	if err == nil {
		t.Errorf("Expect error but none.")
	} else if err.Error() != "failed" {
		t.Errorf("Error not thrown from expected function.")
	}
}

func TestUpgraderRunFailedOnReboot(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.upgradeFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("Windows needs to be restarted")
	}
	tu.rebootFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("failed")
	}
	tu.cleanupFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}

	_, err := Run(tu)
	if err == nil {
		t.Errorf("Expect error but none.")
	} else if err.Error() != "failed" {
		t.Errorf("Error not thrown from expected function.")
	}
}

func TestUpgraderRunFailedOnRetryUpgrade(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.upgradeFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("Windows needs to be restarted")
	}
	tu.rebootFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}
	tu.retryUpgradeFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("failed")
	}
	tu.cleanupFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}

	_, err := Run(tu)
	if err == nil {
		t.Errorf("Expect error but none.")
	} else if err.Error() != "failed" {
		t.Errorf("Error not thrown from expected function.")
	}
}

func TestUpgraderRunSuccessWithoutReboot(t *testing.T) {
	tu := initTestUpgrader(t)

	_, err := Run(tu)
	if err != nil {
		t.Errorf("Unexpected error '%v'.", err)
	}
}

func TestUpgraderRunSuccessWithReboot(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.upgradeFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("Windows needs to be restarted")
	}
	tu.rebootFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}
	tu.retryUpgradeFn = func() (*daisy.Workflow, error) {
		return nil, nil
	}

	_, err := Run(tu)
	if err != nil {
		t.Errorf("Unexpected error '%v'.", err)
	}
}

func TestUpgraderRunFailedWithAutoRollback(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.prepareFn = func() (*daisy.Workflow, error) {
		// Test workaround: let newOSDiskName to be the same as current disk name
		// in order to trigger auto rollback.
		tu.newOSDiskName = testDisk
		return nil, fmt.Errorf("failed")
	}
	tu.AutoRollback = true
	rollbackExecuted := false
	tu.rollbackFn = func() (*daisy.Workflow, error) {
		rollbackExecuted = true
		return nil, nil
	}
	tu.cleanupFn = func() (*daisy.Workflow, error) {
		t.Errorf("Unexpected cleanup.")
		return nil, nil
	}

	_, err := Run(tu)
	if err == nil {
		t.Errorf("Expect error but none.")
	}
	if !rollbackExecuted {
		t.Errorf("Rollback not executed.")
	}
}

func TestUpgraderRunFailedWithAutoRollbackFailed(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.prepareFn = func() (*daisy.Workflow, error) {
		// Test workaround: let newOSDiskName to be the same as current disk name
		// in order to trigger auto rollback.
		tu.newOSDiskName = testDisk
		return nil, fmt.Errorf("failed")
	}
	tu.AutoRollback = true
	rollbackExecuted := false
	tu.rollbackFn = func() (*daisy.Workflow, error) {
		rollbackExecuted = true
		return nil, fmt.Errorf("failed")
	}

	_, err := Run(tu)
	if err == nil {
		t.Errorf("Expect error but none.")
	}
	if !rollbackExecuted {
		t.Errorf("Rollback not executed.")
	}
}

func TestUpgraderRunFailedWithAutoRollbackWithoutNewOSDiskAttached(t *testing.T) {
	tu := initTestUpgrader(t)
	tu.prepareFn = func() (*daisy.Workflow, error) {
		return nil, fmt.Errorf("failed")
	}
	tu.AutoRollback = true
	cleanupExecuted := false
	tu.cleanupFn = func() (*daisy.Workflow, error) {
		cleanupExecuted = true
		return nil, fmt.Errorf("failed")
	}
	_, err := Run(tu)
	if err == nil {
		t.Errorf("Expect error but none.")
	}
	if !cleanupExecuted {
		t.Errorf("Cleanup not executed.")
	}
}

func initTestUpgrader(t *testing.T) *TestUpgrader {
	u := initTest()
	tu := &TestUpgrader{Upgrader: u}
	tu.initFn = func() error {
		computeClient = newTestGCEClient()
		return nil
	}
	tu.prepareFn = func() (workflow *daisy.Workflow, e error) {
		// Test workaround: let newOSDiskName to be the same as current disk name
		// in order to pretend the enw OS disk has been attached.
		tu.newOSDiskName = testDisk
		return nil, nil
	}
	tu.upgradeFn = func() (workflow *daisy.Workflow, e error) {
		return nil, nil
	}
	tu.rebootFn = func() (workflow *daisy.Workflow, e error) {
		t.Errorf("Unexpected reboot.")
		return nil, nil
	}
	tu.retryUpgradeFn = func() (workflow *daisy.Workflow, e error) {
		t.Errorf("Unexpected retryUpgrade.")
		return nil, nil
	}
	tu.cleanupFn = func() (workflow *daisy.Workflow, e error) {
		t.Errorf("Unexpected cleanup.")
		return nil, nil
	}
	tu.rollbackFn = func() (workflow *daisy.Workflow, e error) {
		t.Errorf("Unexpected rollback.")
		return nil, nil
	}
	return tu
}
