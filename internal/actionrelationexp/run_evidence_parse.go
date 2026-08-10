package actionrelationexp

import (
	"bytes"
	"encoding/hex"
	"fmt"
)

func ParseRunEvidencePack(panel, authority string, manifest, data []byte) (RunEvidencePack, error) {
	count := panelRunCounts[panel]
	if count == 0 || len(data) != len(RunEvidenceHeader)+count*RunEvidenceRowSize || !bytes.Equal(data[:len(RunEvidenceHeader)], []byte(RunEvidenceHeader)) {
		return RunEvidencePack{}, fmt.Errorf("invalid run-evidence pack size or header")
	}
	records := make([]RunEvidenceRecord, count)
	zero := make([]byte, 32)
	for index := range records {
		row := data[len(RunEvidenceHeader)+index*RunEvidenceRowSize:][:RunEvidenceRowSize]
		records[index].RunID = hex.EncodeToString(row[:16])
		targets := []*string{
			&records[index].JournalRoot, &records[index].InputRoot, &records[index].DetailRoot,
			&records[index].OperationRoot, &records[index].ChargedRoot, &records[index].StructuralRoot,
		}
		for digestIndex, target := range targets {
			*target = hex.EncodeToString(row[16+digestIndex*32 : 48+digestIndex*32])
		}
		if !bytes.Equal(row[208:240], zero) {
			records[index].WorkTerminal = hex.EncodeToString(row[208:240])
		}
	}
	value, err := BuildRunEvidencePack(panel, authority, records)
	if err != nil || !bytes.Equal(value.Canonical, manifest) || !bytes.Equal(value.File.Data, data) {
		return RunEvidencePack{}, fmt.Errorf("run-evidence manifest does not reconstruct")
	}
	return value, nil
}
