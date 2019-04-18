/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	"testing"

	v1 "k8s.io/api/apps/v1"
)

func TestTransformDaemonSet(t *testing.T) {
	var d v1.DaemonSet
	UnmarshalFile("../../test-data/daemonset.json", &d, t)
	node := transformDaemonSet(&d)

	// Test only the fields that exist in daemonset - the common test will test the other bits
	AssertEqual("available", node.Properties["available"], int64(1), t)
	AssertEqual("current", node.Properties["current"], int64(1), t)
	AssertEqual("desired", node.Properties["desired"], int64(1), t)
	AssertEqual("ready", node.Properties["ready"], int64(1), t)
	AssertEqual("updated", node.Properties["updated"], int64(1), t)
}