// Code generated by protoc-gen-goext. DO NOT EDIT.

package compute

import (
	operation "github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	fieldmaskpb "google.golang.org/protobuf/types/known/fieldmaskpb"
)

func (m *GetPlacementGroupRequest) SetPlacementGroupId(v string) {
	m.PlacementGroupId = v
}

func (m *ListPlacementGroupsRequest) SetFolderId(v string) {
	m.FolderId = v
}

func (m *ListPlacementGroupsRequest) SetPageSize(v int64) {
	m.PageSize = v
}

func (m *ListPlacementGroupsRequest) SetPageToken(v string) {
	m.PageToken = v
}

func (m *ListPlacementGroupsRequest) SetFilter(v string) {
	m.Filter = v
}

func (m *ListPlacementGroupsRequest) SetOrderBy(v string) {
	m.OrderBy = v
}

func (m *ListPlacementGroupsResponse) SetPlacementGroups(v []*PlacementGroup) {
	m.PlacementGroups = v
}

func (m *ListPlacementGroupsResponse) SetNextPageToken(v string) {
	m.NextPageToken = v
}

type CreatePlacementGroupRequest_PlacementStrategy = isCreatePlacementGroupRequest_PlacementStrategy

func (m *CreatePlacementGroupRequest) SetPlacementStrategy(v CreatePlacementGroupRequest_PlacementStrategy) {
	m.PlacementStrategy = v
}

func (m *CreatePlacementGroupRequest) SetFolderId(v string) {
	m.FolderId = v
}

func (m *CreatePlacementGroupRequest) SetName(v string) {
	m.Name = v
}

func (m *CreatePlacementGroupRequest) SetDescription(v string) {
	m.Description = v
}

func (m *CreatePlacementGroupRequest) SetLabels(v map[string]string) {
	m.Labels = v
}

func (m *CreatePlacementGroupRequest) SetSpreadPlacementStrategy(v *SpreadPlacementStrategy) {
	m.PlacementStrategy = &CreatePlacementGroupRequest_SpreadPlacementStrategy{
		SpreadPlacementStrategy: v,
	}
}

func (m *CreatePlacementGroupRequest) SetPartitionPlacementStrategy(v *PartitionPlacementStrategy) {
	m.PlacementStrategy = &CreatePlacementGroupRequest_PartitionPlacementStrategy{
		PartitionPlacementStrategy: v,
	}
}

func (m *CreatePlacementGroupMetadata) SetPlacementGroupId(v string) {
	m.PlacementGroupId = v
}

func (m *UpdatePlacementGroupRequest) SetPlacementGroupId(v string) {
	m.PlacementGroupId = v
}

func (m *UpdatePlacementGroupRequest) SetUpdateMask(v *fieldmaskpb.FieldMask) {
	m.UpdateMask = v
}

func (m *UpdatePlacementGroupRequest) SetName(v string) {
	m.Name = v
}

func (m *UpdatePlacementGroupRequest) SetDescription(v string) {
	m.Description = v
}

func (m *UpdatePlacementGroupRequest) SetLabels(v map[string]string) {
	m.Labels = v
}

func (m *UpdatePlacementGroupMetadata) SetPlacementGroupId(v string) {
	m.PlacementGroupId = v
}

func (m *DeletePlacementGroupRequest) SetPlacementGroupId(v string) {
	m.PlacementGroupId = v
}

func (m *DeletePlacementGroupMetadata) SetPlacementGroupId(v string) {
	m.PlacementGroupId = v
}

func (m *ListPlacementGroupInstancesRequest) SetPlacementGroupId(v string) {
	m.PlacementGroupId = v
}

func (m *ListPlacementGroupInstancesRequest) SetPageSize(v int64) {
	m.PageSize = v
}

func (m *ListPlacementGroupInstancesRequest) SetPageToken(v string) {
	m.PageToken = v
}

func (m *ListPlacementGroupInstancesResponse) SetInstances(v []*Instance) {
	m.Instances = v
}

func (m *ListPlacementGroupInstancesResponse) SetNextPageToken(v string) {
	m.NextPageToken = v
}

func (m *ListPlacementGroupOperationsRequest) SetPlacementGroupId(v string) {
	m.PlacementGroupId = v
}

func (m *ListPlacementGroupOperationsRequest) SetPageSize(v int64) {
	m.PageSize = v
}

func (m *ListPlacementGroupOperationsRequest) SetPageToken(v string) {
	m.PageToken = v
}

func (m *ListPlacementGroupOperationsResponse) SetOperations(v []*operation.Operation) {
	m.Operations = v
}

func (m *ListPlacementGroupOperationsResponse) SetNextPageToken(v string) {
	m.NextPageToken = v
}
