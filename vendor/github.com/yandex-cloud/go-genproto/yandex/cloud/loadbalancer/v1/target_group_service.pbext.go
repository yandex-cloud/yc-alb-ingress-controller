// Code generated by protoc-gen-goext. DO NOT EDIT.

package loadbalancer

import (
	operation "github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	fieldmaskpb "google.golang.org/protobuf/types/known/fieldmaskpb"
)

func (m *GetTargetGroupRequest) SetTargetGroupId(v string) {
	m.TargetGroupId = v
}

func (m *ListTargetGroupsRequest) SetFolderId(v string) {
	m.FolderId = v
}

func (m *ListTargetGroupsRequest) SetPageSize(v int64) {
	m.PageSize = v
}

func (m *ListTargetGroupsRequest) SetPageToken(v string) {
	m.PageToken = v
}

func (m *ListTargetGroupsRequest) SetFilter(v string) {
	m.Filter = v
}

func (m *ListTargetGroupsResponse) SetTargetGroups(v []*TargetGroup) {
	m.TargetGroups = v
}

func (m *ListTargetGroupsResponse) SetNextPageToken(v string) {
	m.NextPageToken = v
}

func (m *CreateTargetGroupRequest) SetFolderId(v string) {
	m.FolderId = v
}

func (m *CreateTargetGroupRequest) SetName(v string) {
	m.Name = v
}

func (m *CreateTargetGroupRequest) SetDescription(v string) {
	m.Description = v
}

func (m *CreateTargetGroupRequest) SetLabels(v map[string]string) {
	m.Labels = v
}

func (m *CreateTargetGroupRequest) SetRegionId(v string) {
	m.RegionId = v
}

func (m *CreateTargetGroupRequest) SetTargets(v []*Target) {
	m.Targets = v
}

func (m *CreateTargetGroupMetadata) SetTargetGroupId(v string) {
	m.TargetGroupId = v
}

func (m *UpdateTargetGroupRequest) SetTargetGroupId(v string) {
	m.TargetGroupId = v
}

func (m *UpdateTargetGroupRequest) SetUpdateMask(v *fieldmaskpb.FieldMask) {
	m.UpdateMask = v
}

func (m *UpdateTargetGroupRequest) SetName(v string) {
	m.Name = v
}

func (m *UpdateTargetGroupRequest) SetDescription(v string) {
	m.Description = v
}

func (m *UpdateTargetGroupRequest) SetLabels(v map[string]string) {
	m.Labels = v
}

func (m *UpdateTargetGroupRequest) SetTargets(v []*Target) {
	m.Targets = v
}

func (m *UpdateTargetGroupMetadata) SetTargetGroupId(v string) {
	m.TargetGroupId = v
}

func (m *DeleteTargetGroupRequest) SetTargetGroupId(v string) {
	m.TargetGroupId = v
}

func (m *DeleteTargetGroupMetadata) SetTargetGroupId(v string) {
	m.TargetGroupId = v
}

func (m *AddTargetsRequest) SetTargetGroupId(v string) {
	m.TargetGroupId = v
}

func (m *AddTargetsRequest) SetTargets(v []*Target) {
	m.Targets = v
}

func (m *AddTargetsMetadata) SetTargetGroupId(v string) {
	m.TargetGroupId = v
}

func (m *RemoveTargetsRequest) SetTargetGroupId(v string) {
	m.TargetGroupId = v
}

func (m *RemoveTargetsRequest) SetTargets(v []*Target) {
	m.Targets = v
}

func (m *RemoveTargetsMetadata) SetTargetGroupId(v string) {
	m.TargetGroupId = v
}

func (m *ListTargetGroupOperationsRequest) SetTargetGroupId(v string) {
	m.TargetGroupId = v
}

func (m *ListTargetGroupOperationsRequest) SetPageSize(v int64) {
	m.PageSize = v
}

func (m *ListTargetGroupOperationsRequest) SetPageToken(v string) {
	m.PageToken = v
}

func (m *ListTargetGroupOperationsResponse) SetOperations(v []*operation.Operation) {
	m.Operations = v
}

func (m *ListTargetGroupOperationsResponse) SetNextPageToken(v string) {
	m.NextPageToken = v
}
