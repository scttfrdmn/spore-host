package quotas

import (
	"strings"
	"testing"
)

func TestGetQuotaFamily(t *testing.T) {
	tests := []struct {
		instanceType string
		want         QuotaFamily
	}{
		{"p5.48xlarge", FamilyP},
		{"p3.2xlarge", FamilyP},
		{"g4dn.xlarge", FamilyG},
		{"g5.2xlarge", FamilyG},
		{"inf1.xlarge", FamilyInf},
		{"inf2.8xlarge", FamilyInf},
		{"trn1.2xlarge", FamilyTrn},
		{"trn1n.32xlarge", FamilyTrn},
		{"f1.2xlarge", FamilyF},
		{"x2gd.xlarge", FamilyX},
		{"x1e.32xlarge", FamilyX},
		{"m7g.large", FamilyStandard},
		{"c6i.xlarge", FamilyStandard},
		{"r6a.2xlarge", FamilyStandard},
		{"t4g.medium", FamilyStandard},
		{"", FamilyStandard},
	}

	for _, tt := range tests {
		t.Run(tt.instanceType, func(t *testing.T) {
			got := GetQuotaFamily(tt.instanceType)
			if got != tt.want {
				t.Errorf("GetQuotaFamily(%q) = %v, want %v", tt.instanceType, got, tt.want)
			}
		})
	}
}

func TestGetVCPUCount(t *testing.T) {
	tests := []struct {
		instanceType string
		want         int32
	}{
		{"t4g.nano", 1},
		{"t4g.micro", 1},
		{"t4g.small", 1},
		{"t4g.medium", 1},
		{"t4g.large", 2},
		{"m7g.xlarge", 4},
		{"c6i.2xlarge", 8},
		{"r6a.4xlarge", 16},
		{"m6i.8xlarge", 32},
		{"c6i.16xlarge", 64},
		{"p5.48xlarge", 192},
		{"unknown.size", 2},
		{"nosize", 2},
	}

	for _, tt := range tests {
		t.Run(tt.instanceType, func(t *testing.T) {
			got := getVCPUCount(tt.instanceType)
			if got != tt.want {
				t.Errorf("getVCPUCount(%q) = %d, want %d", tt.instanceType, got, tt.want)
			}
		})
	}
}

func makeQuotaInfo(onDemand, spot, usage map[QuotaFamily]int32) *QuotaInfo {
	return &QuotaInfo{
		Region:   "us-east-1",
		OnDemand: onDemand,
		Spot:     spot,
		Usage:    usage,
	}
}

func TestCanLaunch(t *testing.T) {
	c := &Client{}

	quotas := makeQuotaInfo(
		map[QuotaFamily]int32{FamilyStandard: 32, FamilyG: 8},
		map[QuotaFamily]int32{FamilyStandard: 64, FamilyG: 0},
		map[QuotaFamily]int32{FamilyStandard: 16, FamilyG: 0},
	)

	tests := []struct {
		name         string
		instanceType string
		vCPUs        int32
		spot         bool
		wantOK       bool
		wantMsgPart  string
	}{
		{
			name:         "on-demand fits",
			instanceType: "m7g.xlarge",
			vCPUs:        4,
			spot:         false,
			wantOK:       true,
		},
		{
			name:         "on-demand exceeds available",
			instanceType: "m7g.xlarge",
			vCPUs:        20, // only 16 available (32-16)
			spot:         false,
			wantOK:       false,
			wantMsgPart:  "Need 20 vCPUs",
		},
		{
			name:         "spot quota zero",
			instanceType: "g5.xlarge",
			vCPUs:        4,
			spot:         true,
			wantOK:       false,
			wantMsgPart:  "quota for G instances is 0",
		},
		{
			name:         "spot fits",
			instanceType: "m7g.xlarge",
			vCPUs:        4,
			spot:         true,
			wantOK:       true,
		},
		{
			name:         "on-demand quota zero",
			instanceType: "m7g.xlarge",
			vCPUs:        4,
			spot:         false,
			wantOK:       false,
			wantMsgPart:  "quota",
		},
	}

	// Override standard on-demand quota to 0 for last test
	tests[4].name = "on-demand quota zero"
	zeroQuotas := makeQuotaInfo(
		map[QuotaFamily]int32{FamilyStandard: 0},
		map[QuotaFamily]int32{FamilyStandard: 64},
		map[QuotaFamily]int32{FamilyStandard: 0},
	)

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := quotas
			if i == 4 {
				q = zeroQuotas
			}
			ok, msg := c.CanLaunch(tt.instanceType, tt.vCPUs, q, tt.spot)
			if ok != tt.wantOK {
				t.Errorf("CanLaunch() ok = %v, want %v (msg: %s)", ok, tt.wantOK, msg)
			}
			if tt.wantMsgPart != "" && !strings.Contains(msg, tt.wantMsgPart) {
				t.Errorf("CanLaunch() msg = %q, want substring %q", msg, tt.wantMsgPart)
			}
		})
	}
}

func TestQuotaIncreaseCommand(t *testing.T) {
	cmd := QuotaIncreaseCommand("us-east-1", FamilyStandard, 256, false)
	if !strings.Contains(cmd, "us-east-1") {
		t.Errorf("command missing region: %s", cmd)
	}
	if !strings.Contains(cmd, "256") {
		t.Errorf("command missing desired value: %s", cmd)
	}

	spotCmd := QuotaIncreaseCommand("eu-west-1", FamilyG, 64, true)
	if !strings.Contains(spotCmd, "eu-west-1") {
		t.Errorf("spot command missing region: %s", spotCmd)
	}
	if !strings.Contains(spotCmd, "64") {
		t.Errorf("spot command missing desired value: %s", spotCmd)
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		input string
		want  int32
	}{
		{"4", 4},
		{"48", 48},
		{"0", 0},
		{"", 0},
		{"abc", 0},
	}
	for _, tt := range tests {
		got := parseInt(tt.input)
		if got != tt.want {
			t.Errorf("parseInt(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
