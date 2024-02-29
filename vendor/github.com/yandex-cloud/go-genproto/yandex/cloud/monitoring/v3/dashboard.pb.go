// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.29.0
// 	protoc        v3.17.3
// source: yandex/cloud/monitoring/v3/dashboard.proto

package monitoring

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

// Dashboard resource.
type Dashboard struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Dashboard ID.
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	// Container id
	//
	// Types that are assignable to Container:
	//
	//	*Dashboard_FolderId
	Container isDashboard_Container `protobuf_oneof:"container"`
	// Creation timestamp.
	CreatedAt *timestamppb.Timestamp `protobuf:"bytes,20,opt,name=created_at,json=createdAt,proto3" json:"created_at,omitempty"`
	// Modification timestamp.
	ModifiedAt *timestamppb.Timestamp `protobuf:"bytes,21,opt,name=modified_at,json=modifiedAt,proto3" json:"modified_at,omitempty"`
	// ID of the user who created the dashboard.
	CreatedBy string `protobuf:"bytes,22,opt,name=created_by,json=createdBy,proto3" json:"created_by,omitempty"`
	// ID of the user who modified the dashboard.
	ModifiedBy string `protobuf:"bytes,23,opt,name=modified_by,json=modifiedBy,proto3" json:"modified_by,omitempty"`
	// Dashboard name.
	Name string `protobuf:"bytes,24,opt,name=name,proto3" json:"name,omitempty"`
	// Dashboard description.
	Description string `protobuf:"bytes,25,opt,name=description,proto3" json:"description,omitempty"`
	// Resource labels as `key:value` pairs.
	Labels map[string]string `protobuf:"bytes,26,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
	// Dashboard title.
	Title string `protobuf:"bytes,27,opt,name=title,proto3" json:"title,omitempty"`
	// List of dashboard widgets.
	Widgets []*Widget `protobuf:"bytes,28,rep,name=widgets,proto3" json:"widgets,omitempty"`
	// Dashboard parametrization.
	Parametrization *Parametrization `protobuf:"bytes,29,opt,name=parametrization,proto3" json:"parametrization,omitempty"`
	// Dashboard etag.
	Etag string `protobuf:"bytes,30,opt,name=etag,proto3" json:"etag,omitempty"`
}

func (x *Dashboard) Reset() {
	*x = Dashboard{}
	if protoimpl.UnsafeEnabled {
		mi := &file_yandex_cloud_monitoring_v3_dashboard_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Dashboard) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Dashboard) ProtoMessage() {}

func (x *Dashboard) ProtoReflect() protoreflect.Message {
	mi := &file_yandex_cloud_monitoring_v3_dashboard_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Dashboard.ProtoReflect.Descriptor instead.
func (*Dashboard) Descriptor() ([]byte, []int) {
	return file_yandex_cloud_monitoring_v3_dashboard_proto_rawDescGZIP(), []int{0}
}

func (x *Dashboard) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (m *Dashboard) GetContainer() isDashboard_Container {
	if m != nil {
		return m.Container
	}
	return nil
}

func (x *Dashboard) GetFolderId() string {
	if x, ok := x.GetContainer().(*Dashboard_FolderId); ok {
		return x.FolderId
	}
	return ""
}

func (x *Dashboard) GetCreatedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.CreatedAt
	}
	return nil
}

func (x *Dashboard) GetModifiedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.ModifiedAt
	}
	return nil
}

func (x *Dashboard) GetCreatedBy() string {
	if x != nil {
		return x.CreatedBy
	}
	return ""
}

func (x *Dashboard) GetModifiedBy() string {
	if x != nil {
		return x.ModifiedBy
	}
	return ""
}

func (x *Dashboard) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Dashboard) GetDescription() string {
	if x != nil {
		return x.Description
	}
	return ""
}

func (x *Dashboard) GetLabels() map[string]string {
	if x != nil {
		return x.Labels
	}
	return nil
}

func (x *Dashboard) GetTitle() string {
	if x != nil {
		return x.Title
	}
	return ""
}

func (x *Dashboard) GetWidgets() []*Widget {
	if x != nil {
		return x.Widgets
	}
	return nil
}

func (x *Dashboard) GetParametrization() *Parametrization {
	if x != nil {
		return x.Parametrization
	}
	return nil
}

func (x *Dashboard) GetEtag() string {
	if x != nil {
		return x.Etag
	}
	return ""
}

type isDashboard_Container interface {
	isDashboard_Container()
}

type Dashboard_FolderId struct {
	// Folder ID.
	FolderId string `protobuf:"bytes,3,opt,name=folder_id,json=folderId,proto3,oneof"`
}

func (*Dashboard_FolderId) isDashboard_Container() {}

var File_yandex_cloud_monitoring_v3_dashboard_proto protoreflect.FileDescriptor

var file_yandex_cloud_monitoring_v3_dashboard_proto_rawDesc = []byte{
	0x0a, 0x2a, 0x79, 0x61, 0x6e, 0x64, 0x65, 0x78, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2f, 0x6d,
	0x6f, 0x6e, 0x69, 0x74, 0x6f, 0x72, 0x69, 0x6e, 0x67, 0x2f, 0x76, 0x33, 0x2f, 0x64, 0x61, 0x73,
	0x68, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x1a, 0x79, 0x61,
	0x6e, 0x64, 0x65, 0x78, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2e, 0x6d, 0x6f, 0x6e, 0x69, 0x74,
	0x6f, 0x72, 0x69, 0x6e, 0x67, 0x2e, 0x76, 0x33, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74,
	0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x30, 0x79, 0x61, 0x6e, 0x64, 0x65,
	0x78, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2f, 0x6d, 0x6f, 0x6e, 0x69, 0x74, 0x6f, 0x72, 0x69,
	0x6e, 0x67, 0x2f, 0x76, 0x33, 0x2f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x7a,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x27, 0x79, 0x61, 0x6e,
	0x64, 0x65, 0x78, 0x2f, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2f, 0x6d, 0x6f, 0x6e, 0x69, 0x74, 0x6f,
	0x72, 0x69, 0x6e, 0x67, 0x2f, 0x76, 0x33, 0x2f, 0x77, 0x69, 0x64, 0x67, 0x65, 0x74, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x22, 0xfa, 0x04, 0x0a, 0x09, 0x44, 0x61, 0x73, 0x68, 0x62, 0x6f, 0x61,
	0x72, 0x64, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x02,
	0x69, 0x64, 0x12, 0x1d, 0x0a, 0x09, 0x66, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x5f, 0x69, 0x64, 0x18,
	0x03, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x08, 0x66, 0x6f, 0x6c, 0x64, 0x65, 0x72, 0x49,
	0x64, 0x12, 0x39, 0x0a, 0x0a, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18,
	0x14, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d,
	0x70, 0x52, 0x09, 0x63, 0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x41, 0x74, 0x12, 0x3b, 0x0a, 0x0b,
	0x6d, 0x6f, 0x64, 0x69, 0x66, 0x69, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x15, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x0a, 0x6d,
	0x6f, 0x64, 0x69, 0x66, 0x69, 0x65, 0x64, 0x41, 0x74, 0x12, 0x1d, 0x0a, 0x0a, 0x63, 0x72, 0x65,
	0x61, 0x74, 0x65, 0x64, 0x5f, 0x62, 0x79, 0x18, 0x16, 0x20, 0x01, 0x28, 0x09, 0x52, 0x09, 0x63,
	0x72, 0x65, 0x61, 0x74, 0x65, 0x64, 0x42, 0x79, 0x12, 0x1f, 0x0a, 0x0b, 0x6d, 0x6f, 0x64, 0x69,
	0x66, 0x69, 0x65, 0x64, 0x5f, 0x62, 0x79, 0x18, 0x17, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0a, 0x6d,
	0x6f, 0x64, 0x69, 0x66, 0x69, 0x65, 0x64, 0x42, 0x79, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x18, 0x18, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x20, 0x0a,
	0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x19, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12,
	0x49, 0x0a, 0x06, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x18, 0x1a, 0x20, 0x03, 0x28, 0x0b, 0x32,
	0x31, 0x2e, 0x79, 0x61, 0x6e, 0x64, 0x65, 0x78, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2e, 0x6d,
	0x6f, 0x6e, 0x69, 0x74, 0x6f, 0x72, 0x69, 0x6e, 0x67, 0x2e, 0x76, 0x33, 0x2e, 0x44, 0x61, 0x73,
	0x68, 0x62, 0x6f, 0x61, 0x72, 0x64, 0x2e, 0x4c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x45, 0x6e, 0x74,
	0x72, 0x79, 0x52, 0x06, 0x6c, 0x61, 0x62, 0x65, 0x6c, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x69,
	0x74, 0x6c, 0x65, 0x18, 0x1b, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x69, 0x74, 0x6c, 0x65,
	0x12, 0x3c, 0x0a, 0x07, 0x77, 0x69, 0x64, 0x67, 0x65, 0x74, 0x73, 0x18, 0x1c, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x22, 0x2e, 0x79, 0x61, 0x6e, 0x64, 0x65, 0x78, 0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64,
	0x2e, 0x6d, 0x6f, 0x6e, 0x69, 0x74, 0x6f, 0x72, 0x69, 0x6e, 0x67, 0x2e, 0x76, 0x33, 0x2e, 0x57,
	0x69, 0x64, 0x67, 0x65, 0x74, 0x52, 0x07, 0x77, 0x69, 0x64, 0x67, 0x65, 0x74, 0x73, 0x12, 0x55,
	0x0a, 0x0f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x7a, 0x61, 0x74, 0x69, 0x6f,
	0x6e, 0x18, 0x1d, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x2b, 0x2e, 0x79, 0x61, 0x6e, 0x64, 0x65, 0x78,
	0x2e, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2e, 0x6d, 0x6f, 0x6e, 0x69, 0x74, 0x6f, 0x72, 0x69, 0x6e,
	0x67, 0x2e, 0x76, 0x33, 0x2e, 0x50, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x7a, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x52, 0x0f, 0x70, 0x61, 0x72, 0x61, 0x6d, 0x65, 0x74, 0x72, 0x69, 0x7a,
	0x61, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x65, 0x74, 0x61, 0x67, 0x18, 0x1e, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x04, 0x65, 0x74, 0x61, 0x67, 0x1a, 0x39, 0x0a, 0x0b, 0x4c, 0x61, 0x62,
	0x65, 0x6c, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05, 0x76, 0x61,
	0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65,
	0x3a, 0x02, 0x38, 0x01, 0x42, 0x0b, 0x0a, 0x09, 0x63, 0x6f, 0x6e, 0x74, 0x61, 0x69, 0x6e, 0x65,
	0x72, 0x42, 0x6b, 0x0a, 0x1e, 0x79, 0x61, 0x6e, 0x64, 0x65, 0x78, 0x2e, 0x63, 0x6c, 0x6f, 0x75,
	0x64, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x6d, 0x6f, 0x6e, 0x69, 0x74, 0x6f, 0x72, 0x69, 0x6e, 0x67,
	0x2e, 0x76, 0x33, 0x5a, 0x49, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x79, 0x61, 0x6e, 0x64, 0x65, 0x78, 0x2d, 0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2f, 0x67, 0x6f, 0x2d,
	0x67, 0x65, 0x6e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x79, 0x61, 0x6e, 0x64, 0x65, 0x78, 0x2f,
	0x63, 0x6c, 0x6f, 0x75, 0x64, 0x2f, 0x6d, 0x6f, 0x6e, 0x69, 0x74, 0x6f, 0x72, 0x69, 0x6e, 0x67,
	0x2f, 0x76, 0x33, 0x3b, 0x6d, 0x6f, 0x6e, 0x69, 0x74, 0x6f, 0x72, 0x69, 0x6e, 0x67, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_yandex_cloud_monitoring_v3_dashboard_proto_rawDescOnce sync.Once
	file_yandex_cloud_monitoring_v3_dashboard_proto_rawDescData = file_yandex_cloud_monitoring_v3_dashboard_proto_rawDesc
)

func file_yandex_cloud_monitoring_v3_dashboard_proto_rawDescGZIP() []byte {
	file_yandex_cloud_monitoring_v3_dashboard_proto_rawDescOnce.Do(func() {
		file_yandex_cloud_monitoring_v3_dashboard_proto_rawDescData = protoimpl.X.CompressGZIP(file_yandex_cloud_monitoring_v3_dashboard_proto_rawDescData)
	})
	return file_yandex_cloud_monitoring_v3_dashboard_proto_rawDescData
}

var file_yandex_cloud_monitoring_v3_dashboard_proto_msgTypes = make([]protoimpl.MessageInfo, 2)
var file_yandex_cloud_monitoring_v3_dashboard_proto_goTypes = []interface{}{
	(*Dashboard)(nil),             // 0: yandex.cloud.monitoring.v3.Dashboard
	nil,                           // 1: yandex.cloud.monitoring.v3.Dashboard.LabelsEntry
	(*timestamppb.Timestamp)(nil), // 2: google.protobuf.Timestamp
	(*Widget)(nil),                // 3: yandex.cloud.monitoring.v3.Widget
	(*Parametrization)(nil),       // 4: yandex.cloud.monitoring.v3.Parametrization
}
var file_yandex_cloud_monitoring_v3_dashboard_proto_depIdxs = []int32{
	2, // 0: yandex.cloud.monitoring.v3.Dashboard.created_at:type_name -> google.protobuf.Timestamp
	2, // 1: yandex.cloud.monitoring.v3.Dashboard.modified_at:type_name -> google.protobuf.Timestamp
	1, // 2: yandex.cloud.monitoring.v3.Dashboard.labels:type_name -> yandex.cloud.monitoring.v3.Dashboard.LabelsEntry
	3, // 3: yandex.cloud.monitoring.v3.Dashboard.widgets:type_name -> yandex.cloud.monitoring.v3.Widget
	4, // 4: yandex.cloud.monitoring.v3.Dashboard.parametrization:type_name -> yandex.cloud.monitoring.v3.Parametrization
	5, // [5:5] is the sub-list for method output_type
	5, // [5:5] is the sub-list for method input_type
	5, // [5:5] is the sub-list for extension type_name
	5, // [5:5] is the sub-list for extension extendee
	0, // [0:5] is the sub-list for field type_name
}

func init() { file_yandex_cloud_monitoring_v3_dashboard_proto_init() }
func file_yandex_cloud_monitoring_v3_dashboard_proto_init() {
	if File_yandex_cloud_monitoring_v3_dashboard_proto != nil {
		return
	}
	file_yandex_cloud_monitoring_v3_parametrization_proto_init()
	file_yandex_cloud_monitoring_v3_widget_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_yandex_cloud_monitoring_v3_dashboard_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Dashboard); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_yandex_cloud_monitoring_v3_dashboard_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*Dashboard_FolderId)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_yandex_cloud_monitoring_v3_dashboard_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   2,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_yandex_cloud_monitoring_v3_dashboard_proto_goTypes,
		DependencyIndexes: file_yandex_cloud_monitoring_v3_dashboard_proto_depIdxs,
		MessageInfos:      file_yandex_cloud_monitoring_v3_dashboard_proto_msgTypes,
	}.Build()
	File_yandex_cloud_monitoring_v3_dashboard_proto = out.File
	file_yandex_cloud_monitoring_v3_dashboard_proto_rawDesc = nil
	file_yandex_cloud_monitoring_v3_dashboard_proto_goTypes = nil
	file_yandex_cloud_monitoring_v3_dashboard_proto_depIdxs = nil
}
