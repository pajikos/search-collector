/*
IBM Confidential
OCO Source Materials
5737-E67
(C) Copyright IBM Corporation 2019 All Rights Reserved
The source code for this program is not published or otherwise divested of its trade secrets, irrespective of what has been deposited with the U.S. Copyright Office.
*/

package transforms

import (
	appDeployable "github.ibm.com/IBMMulticloudPlatform/deployable/pkg/apis/app/v1alpha1"
)

type AppDeployableResource struct {
	*appDeployable.Deployable
}

func (d AppDeployableResource) BuildNode() Node {
	node := transformCommon(d) // Start off with the common properties

	// Extract the properties specific to this type
	node.Properties["kind"] = "Deployable"
	node.Properties["apigroup"] = "app.ibm.com"
	//TODO: Add properties, TEMPLATE-KIND   TEMPLATE-APIVERSION    AGE   STATUS
	return node
}

func (d AppDeployableResource) BuildEdges(ns NodeStore) []Edge {
	ret := []Edge{}
	UID := prefixedUID(d.UID)

	nodeInfo := NodeInfo{NameSpace: d.Namespace, UID: UID, EdgeType: "promotedTo", Kind: d.Kind, Name: d.Name}
	channelMap := make(map[string]struct{})
	//promotedTo edges
	if d.Spec.Channels != nil {
		for _, channel := range d.Spec.Channels {
			channelMap[channel] = struct{}{}
		}
		ret = append(ret, edgesByDestinationName(channelMap, ret, "Channel", nodeInfo, ns)...)
	}

	//refersTo edges
	//Builds edges between deployable and placement rule
	if d.Spec.Placement != nil && d.Spec.Placement.PlacementRef != nil && d.Spec.Placement.PlacementRef.Name != "" {
		nodeInfo.EdgeType = "refersTo"
		placementRuleMap := make(map[string]struct{})
		placementRuleMap[d.Spec.Placement.PlacementRef.Name] = struct{}{}
		ret = append(ret, edgesByDestinationName(placementRuleMap, ret, "PlacementRule", nodeInfo, ns)...)
	}
	//deployer subscriber edges
	ret = append(ret, edgesByDeployerSubscriber(nodeInfo, ns)...)
	return ret
}