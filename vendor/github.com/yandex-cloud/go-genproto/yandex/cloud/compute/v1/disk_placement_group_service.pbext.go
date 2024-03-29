// Code generated by protoc-gen-goext. DO NOT EDIT.

package compute

import (
	operation "github.com/yandex-cloud/go-genproto/yandex/cloud/operation"
	fieldmaskpb "google.golang.org/protobuf/types/known/fieldmaskpb"
)

func (m *GetDiskPlacementGroupRequest) SetDiskPlacementGroupId(v string) {
	m.DiskPlacementGroupId = v
}

func (m *ListDiskPlacementGroupsRequest) SetFolderId(v string) {
	m.FolderId = v
}

func (m *ListDiskPlacementGroupsRequest) SetPageSize(v int64) {
	m.PageSize = v
}

func (m *ListDiskPlacementGroupsRequest) SetPageToken(v string) {
	m.PageToken = v
}

func (m *ListDiskPlacementGroupsRequest) SetFilter(v string) {
	m.Filter = v
}

func (m *ListDiskPlacementGroupsRequest) SetOrderBy(v string) {
	m.OrderBy = v
}

func (m *ListDiskPlacementGroupsResponse) SetDiskPlacementGroups(v []*DiskPlacementGroup) {
	m.DiskPlacementGroups = v
}

func (m *ListDiskPlacementGroupsResponse) SetNextPageToken(v string) {
	m.NextPageToken = v
}

type CreateDiskPlacementGroupRequest_PlacementStrategy = isCreateDiskPlacementGroupRequest_PlacementStrategy

func (m *CreateDiskPlacementGroupRequest) SetPlacementStrategy(v CreateDiskPlacementGroupRequest_PlacementStrategy) {
	m.PlacementStrategy = v
}

func (m *CreateDiskPlacementGroupRequest) SetFolderId(v string) {
	m.FolderId = v
}

func (m *CreateDiskPlacementGroupRequest) SetName(v string) {
	m.Name = v
}

func (m *CreateDiskPlacementGroupRequest) SetDescription(v string) {
	m.Description = v
}

func (m *CreateDiskPlacementGroupRequest) SetLabels(v map[string]string) {
	m.Labels = v
}

func (m *CreateDiskPlacementGroupRequest) SetZoneId(v string) {
	m.ZoneId = v
}

func (m *CreateDiskPlacementGroupRequest) SetSpreadPlacementStrategy(v *DiskSpreadPlacementStrategy) {
	m.PlacementStrategy = &CreateDiskPlacementGroupRequest_SpreadPlacementStrategy{
		SpreadPlacementStrategy: v,
	}
}

func (m *CreateDiskPlacementGroupRequest) SetPartitionPlacementStrategy(v *DiskPartitionPlacementStrategy) {
	m.PlacementStrategy = &CreateDiskPlacementGroupRequest_PartitionPlacementStrategy{
		PartitionPlacementStrategy: v,
	}
}

func (m *CreateDiskPlacementGroupMetadata) SetDiskPlacementGroupId(v string) {
	m.DiskPlacementGroupId = v
}

func (m *UpdateDiskPlacementGroupRequest) SetDiskPlacementGroupId(v string) {
	m.DiskPlacementGroupId = v
}

func (m *UpdateDiskPlacementGroupRequest) SetUpdateMask(v *fieldmaskpb.FieldMask) {
	m.UpdateMask = v
}

func (m *UpdateDiskPlacementGroupRequest) SetName(v string) {
	m.Name = v
}

func (m *UpdateDiskPlacementGroupRequest) SetDescription(v string) {
	m.Description = v
}

func (m *UpdateDiskPlacementGroupRequest) SetLabels(v map[string]string) {
	m.Labels = v
}

func (m *UpdateDiskPlacementGroupMetadata) SetDiskPlacementGroupId(v string) {
	m.DiskPlacementGroupId = v
}

func (m *DeleteDiskPlacementGroupRequest) SetDiskPlacementGroupId(v string) {
	m.DiskPlacementGroupId = v
}

func (m *DeleteDiskPlacementGroupMetadata) SetDiskPlacementGroupId(v string) {
	m.DiskPlacementGroupId = v
}

func (m *ListDiskPlacementGroupDisksRequest) SetDiskPlacementGroupId(v string) {
	m.DiskPlacementGroupId = v
}

func (m *ListDiskPlacementGroupDisksRequest) SetPageSize(v int64) {
	m.PageSize = v
}

func (m *ListDiskPlacementGroupDisksRequest) SetPageToken(v string) {
	m.PageToken = v
}

func (m *ListDiskPlacementGroupDisksResponse) SetDisks(v []*Disk) {
	m.Disks = v
}

func (m *ListDiskPlacementGroupDisksResponse) SetNextPageToken(v string) {
	m.NextPageToken = v
}

func (m *ListDiskPlacementGroupOperationsRequest) SetDiskPlacementGroupId(v string) {
	m.DiskPlacementGroupId = v
}

func (m *ListDiskPlacementGroupOperationsRequest) SetPageSize(v int64) {
	m.PageSize = v
}

func (m *ListDiskPlacementGroupOperationsRequest) SetPageToken(v string) {
	m.PageToken = v
}

func (m *ListDiskPlacementGroupOperationsResponse) SetOperations(v []*operation.Operation) {
	m.Operations = v
}

func (m *ListDiskPlacementGroupOperationsResponse) SetNextPageToken(v string) {
	m.NextPageToken = v
}
