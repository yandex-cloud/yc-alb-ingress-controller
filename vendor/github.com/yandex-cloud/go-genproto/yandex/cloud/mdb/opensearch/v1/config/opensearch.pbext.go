// Code generated by protoc-gen-goext. DO NOT EDIT.

package opensearch

import (
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func (m *OpenSearchConfig2) SetMaxClauseCount(v *wrapperspb.Int64Value) {
	m.MaxClauseCount = v
}

func (m *OpenSearchConfig2) SetFielddataCacheSize(v string) {
	m.FielddataCacheSize = v
}

func (m *OpenSearchConfig2) SetReindexRemoteWhitelist(v string) {
	m.ReindexRemoteWhitelist = v
}

func (m *OpenSearchConfigSet2) SetEffectiveConfig(v *OpenSearchConfig2) {
	m.EffectiveConfig = v
}

func (m *OpenSearchConfigSet2) SetUserConfig(v *OpenSearchConfig2) {
	m.UserConfig = v
}

func (m *OpenSearchConfigSet2) SetDefaultConfig(v *OpenSearchConfig2) {
	m.DefaultConfig = v
}
