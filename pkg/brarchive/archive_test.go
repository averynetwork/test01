package brarchive

import (
    "reflect"
    "testing"
)

func TestSerializeDeserialize(t *testing.T) {
    tests := []struct {
        name    string
        entries map[string]string
        wantErr bool
    }{
        {
            name: "single file",
            entries: map[string]string{
                "test.txt": "Hello, World!",
            },
            wantErr: false,
        },
        {
            name: "multiple files",
            entries: map[string]string{
                "file1.txt": "Content 1",
                "file2.txt": "Content 2",
                "file3.txt": "Content 3",
            },
            wantErr: false,
        },
        {
            name: "long filename",
            entries: map[string]string{
                "this_is_a_very_long_filename_that_is_still_under_247_bytes.txt": "Content",
            },
            wantErr: false,
        },
        {
            name:    "empty entries",
            entries: map[string]string{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Serialize
            serialized, err := Serialize(tt.entries)
            if (err != nil) != tt.wantErr {
                t.Errorf("Serialize() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if tt.wantErr {
                return
            }

            // Deserialize
            deserialized, err := Deserialize(serialized)
            if err != nil {
                t.Errorf("Deserialize() error = %v", err)
                return
            }

            // Compare
            if !reflect.DeepEqual(deserialized, tt.entries) {
                t.Errorf("Deserialize() = %v, want %v", deserialized, tt.entries)
            }
        })
    }
}

func TestInvalidData(t *testing.T) {
    tests := []struct {
        name    string
        data    []byte
        wantErr bool
    }{
        {
            name:    "empty data",
            data:    []byte{},
            wantErr: true,
        },
        {
            name:    "invalid magic",
            data:    []byte{0, 1, 2, 3, 4, 5, 6, 7, 1, 0, 0, 0, 1, 0, 0, 0},
            wantErr: true,
        },
        {
            name:    "truncated data",
            data:    []byte{0x7D, 0x27, 0x25, 0xB1, 0xA0, 0x52, 0x70, 0x26, 0x01, 0x00, 0x00, 0x00},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := Deserialize(tt.data)
            if (err != nil) != tt.wantErr {
                t.Errorf("Deserialize() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
