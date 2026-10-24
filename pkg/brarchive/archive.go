package brarchive

import (
    "bytes"
    "encoding/binary"
    "fmt"
    "sort"
)

const (
    version           = 1
    magic            uint64 = 0x267052A0B125277D
    entryNameLenMax  int    = 247
)

// Header represents the archive header
type Header struct {
    Magic   uint64
    Entries uint32
    Version uint32
}

// EntryDescriptor represents a file entry in the archive
type EntryDescriptor struct {
    Name           string
    ContentsOffset uint32
    ContentsLen    uint32
}

// Serialize encodes a map of filename to content into brarchive format (Rust compatible)
func Serialize(entries map[string]string) ([]byte, error) {
    if len(entries) == 0 {
        return nil, fmt.Errorf("no entries to encode")
    }

    var buf bytes.Buffer

    // Sort entries for deterministic output
    keys := make([]string, 0, len(entries))
    for k := range entries {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    // Calculate offsets first
    descriptors := make([]EntryDescriptor, 0, len(entries))
    var currentOffset uint32 = 0

    for _, key := range keys {
        content := entries[key]
        descriptors = append(descriptors, EntryDescriptor{
            Name:           key,
            ContentsOffset: currentOffset,
            ContentsLen:    uint32(len(content)),
        })
        currentOffset += uint32(len(content))
    }

    // Write header: Magic (8 bytes) + Entries (4 bytes) + Version (4 bytes)
    if err := binary.Write(&buf, binary.LittleEndian, magic); err != nil {
        return nil, fmt.Errorf("failed to write magic: %v", err)
    }

    if err := binary.Write(&buf, binary.LittleEndian, uint32(len(entries))); err != nil {
        return nil, fmt.Errorf("failed to write entry count: %v", err)
    }

    if err := binary.Write(&buf, binary.LittleEndian, uint32(version)); err != nil {
        return nil, fmt.Errorf("failed to write version: %v", err)
    }

    // Write entry descriptors
    for _, desc := range descriptors {
        // Write filename length (1 byte)
        nameBytes := []byte(desc.Name)
        if len(nameBytes) > entryNameLenMax {
            return nil, fmt.Errorf("entry name too long: %s (%d bytes, max %d)", 
                desc.Name, len(nameBytes), entryNameLenMax)
        }
        
        if err := binary.Write(&buf, binary.LittleEndian, uint8(len(nameBytes))); err != nil {
            return nil, fmt.Errorf("failed to write name length: %v", err)
        }

        // Write filename (fixed 247 bytes, padded with zeros)
        nameBuf := make([]byte, entryNameLenMax)
        copy(nameBuf, nameBytes)
        if _, err := buf.Write(nameBuf); err != nil {
            return nil, fmt.Errorf("failed to write name: %v", err)
        }

        // Write contents offset (4 bytes)
        if err := binary.Write(&buf, binary.LittleEndian, desc.ContentsOffset); err != nil {
            return nil, fmt.Errorf("failed to write offset: %v", err)
        }

        // Write contents length (4 bytes)
        if err := binary.Write(&buf, binary.LittleEndian, desc.ContentsLen); err != nil {
            return nil, fmt.Errorf("failed to write content length: %v", err)
        }
    }

    // Write contents
    for _, key := range keys {
        content := entries[key]
        if _, err := buf.WriteString(content); err != nil {
            return nil, fmt.Errorf("failed to write content: %v", err)
        }
    }

    return buf.Bytes(), nil
}

// Deserialize decodes brarchive format into a map of filename to content
func Deserialize(data []byte) (map[string]string, error) {
    if len(data) < 16 { // Magic(8) + Entries(4) + Version(4)
        return nil, fmt.Errorf("data too short: %d bytes", len(data))
    }

    buf := bytes.NewReader(data)
    result := make(map[string]string)

    // Read and verify magic
    var magicNum uint64
    if err := binary.Read(buf, binary.LittleEndian, &magicNum); err != nil {
        return nil, fmt.Errorf("failed to read magic: %v", err)
    }

    if magicNum != magic {
        return nil, fmt.Errorf("invalid magic number: expected 0x%x, got 0x%x", magic, magicNum)
    }

    // Read entry count
    var entries uint32
    if err := binary.Read(buf, binary.LittleEndian, &entries); err != nil {
        return nil, fmt.Errorf("failed to read entry count: %v", err)
    }

    // Read version
    var ver uint32
    if err := binary.Read(buf, binary.LittleEndian, &ver); err != nil {
        return nil, fmt.Errorf("failed to read version: %v", err)
    }

    if ver != version {
        return nil, fmt.Errorf("unsupported version: %d, expected %d", ver, version)
    }

    // Read entry descriptors
    descriptors := make([]EntryDescriptor, 0, entries)
    for i := uint32(0); i < entries; i++ {
        // Read filename length
        var nameLen uint8
        if err := binary.Read(buf, binary.LittleEndian, &nameLen); err != nil {
            return nil, fmt.Errorf("failed to read name length at entry %d: %v", i, err)
        }

        if nameLen > uint8(entryNameLenMax) {
            return nil, fmt.Errorf("invalid name length at entry %d: %d", i, nameLen)
        }

        // Read filename (fixed 247 bytes)
        nameBuf := make([]byte, entryNameLenMax)
        n, err := buf.Read(nameBuf)
        if err != nil || n != entryNameLenMax {
            return nil, fmt.Errorf("failed to read name at entry %d: %v", i, err)
        }

        // Extract actual filename (up to nameLen bytes)
        fileName := string(nameBuf[:nameLen])

        // Read contents offset
        var offset uint32
        if err := binary.Read(buf, binary.LittleEndian, &offset); err != nil {
            return nil, fmt.Errorf("failed to read offset at entry %d: %v", i, err)
        }

        // Read contents length
        var contentLen uint32
        if err := binary.Read(buf, binary.LittleEndian, &contentLen); err != nil {
            return nil, fmt.Errorf("failed to read content length at entry %d: %v", i, err)
        }

        descriptors = append(descriptors, EntryDescriptor{
            Name:           fileName,
            ContentsOffset: offset,
            ContentsLen:    contentLen,
        })
    }

    // Save current position (start of content section)
    contentStart, err := buf.Seek(0, 1)
    if err != nil {
        return nil, fmt.Errorf("failed to get content start: %v", err)
    }

    // Read contents
    for _, desc := range descriptors {
        // Seek to content position
        if _, err := buf.Seek(contentStart+int64(desc.ContentsOffset), 0); err != nil {
            return nil, fmt.Errorf("failed to seek to content for %s: %v", desc.Name, err)
        }

        // Read content
        contentBuf := make([]byte, desc.ContentsLen)
        n, err := buf.Read(contentBuf)
        if err != nil || uint32(n) != desc.ContentsLen {
            return nil, fmt.Errorf("failed to read content for %s: %v", desc.Name, err)
        }

        result[desc.Name] = string(contentBuf)
    }

    return result, nil
}
