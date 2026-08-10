package actionrelationfixturecore

import "testing"

func TestEveryTrainingCoreHasTwoVerifiedNameBanks(t *testing.T) {
	cases, err := Training()
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range cases {
		views, err := Views(testCase)
		if err != nil {
			t.Fatalf("case %d: %v", testCase.Ordinal, err)
		}
		if len(views) != 2 || views[0].Digest == views[1].Digest || views[0].SemanticWorldDigest != views[1].SemanticWorldDigest || views[0].Bank != 0 || views[1].Bank != 1 {
			t.Fatalf("case %d views=%+v", testCase.Ordinal, views)
		}
	}
}
