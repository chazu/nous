package actionrelationscore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

const maximumPanelSummaryBytes = int64(64 * 1024 * 1024)

func MarshalPanelSummary(summary PanelSummary) ([]byte, error) {
	if err := VerifyPanelSummary(summary); err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(summary)
	if err != nil || int64(len(canonical)) > maximumPanelSummaryBytes {
		return nil, fmt.Errorf("panel summary exceeds isolated result cap")
	}
	return canonical, nil
}

func ParsePanelSummary(reader io.Reader, size int64) (PanelSummary, error) {
	if size < 1 || size > maximumPanelSummaryBytes {
		return PanelSummary{}, fmt.Errorf("invalid isolated panel summary size")
	}
	data, err := io.ReadAll(io.LimitReader(reader, size+1))
	if err != nil || int64(len(data)) != size {
		return PanelSummary{}, fmt.Errorf("invalid isolated panel summary bytes")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var summary PanelSummary
	if err := decoder.Decode(&summary); err != nil || decoder.Decode(&struct{}{}) != io.EOF || VerifyPanelSummary(summary) != nil {
		return PanelSummary{}, fmt.Errorf("invalid isolated panel summary")
	}
	want, err := json.Marshal(summary)
	if err != nil || !bytes.Equal(want, data) {
		return PanelSummary{}, fmt.Errorf("noncanonical isolated panel summary")
	}
	return summary, nil
}
