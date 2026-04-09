package pricing

import "strings"

// ebsRates maps EBS volume type to $/GB-month (us-east-1 base rates)
var ebsRates = map[string]float64{
	"gp3": 0.08,  // $0.08/GB-month base
	"gp2": 0.10,  // $0.10/GB-month
	"io1": 0.125, // $0.125/GB-month + $0.065/IOPS
	"io2": 0.065, // $0.065/GB-month + $0.065/IOPS
	"st1": 0.045, // $0.045/GB-month
	"sc1": 0.018, // $0.018/GB-month
}

// ebsRegionMultipliers adjusts EBS rates relative to us-east-1
var ebsRegionMultipliers = map[string]float64{
	"us-east-1":      1.0,
	"us-east-2":      1.0,
	"us-west-1":      1.1,
	"us-west-2":      1.0,
	"eu-west-1":      1.1,
	"eu-west-2":      1.12,
	"eu-central-1":   1.15,
	"ap-southeast-1": 1.15,
	"ap-southeast-2": 1.15,
	"ap-northeast-1": 1.2,
}

// GetEBSMonthlyRate returns $/GB-month for an EBS volume type in a region.
// For gp3, iops > 3000 incurs additional $0.004/IOPS/month.
// For io1/io2, iops incurs $0.065/IOPS/month.
func GetEBSMonthlyRate(region, volumeType string, sizeGB, iops int) float64 {
	volumeType = strings.ToLower(strings.TrimSpace(volumeType))
	if volumeType == "" {
		volumeType = "gp2"
	}

	baseRate, ok := ebsRates[volumeType]
	if !ok {
		baseRate = ebsRates["gp2"]
	}

	// Provisioned IOPS surcharge (per-volume, converted to per-GB for consistency)
	iopsMonthly := 0.0
	if sizeGB > 0 {
		switch volumeType {
		case "gp3":
			if iops > 3000 {
				iopsMonthly = float64(iops-3000) * 0.004 / float64(sizeGB)
			}
		case "io1", "io2":
			if iops > 0 {
				iopsMonthly = float64(iops) * 0.065 / float64(sizeGB)
			}
		}
	}

	totalRate := baseRate + iopsMonthly

	// Regional multiplier
	var multiplier float64
	if m, ok := ebsRegionMultipliers[strings.ToLower(region)]; ok {
		multiplier = m
	} else {
		multiplier = 1.1 // Conservative estimate for unknown regions
	}

	return totalRate * multiplier
}

// GetIPv4HourlyRate returns the hourly rate per public IPv4 address ($0.005/hr as of 2024).
func GetIPv4HourlyRate() float64 {
	return 0.005
}
