// Code generated by protoc-gen-goext. DO NOT EDIT.

package test

type FilePointer_FilePointer = isFilePointer_FilePointer

func (m *FilePointer) SetFilePointer(v FilePointer_FilePointer) {
	m.FilePointer = v
}

func (m *FilePointer) SetObjectStorage(v *ObjectStorage) {
	m.FilePointer = &FilePointer_ObjectStorage{
		ObjectStorage: v,
	}
}
