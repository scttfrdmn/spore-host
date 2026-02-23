package pipeline

import (
	"fmt"
	"strings"
)

// RenderGraph renders an ASCII DAG visualization of the pipeline
func (p *Pipeline) RenderGraph() (string, error) {
	order, err := p.GetTopologicalOrder()
	if err != nil {
		return "", err
	}

	// Build dependency map (stage -> stages that depend on it)
	downstreamMap := make(map[string][]string)
	for _, stage := range p.Stages {
		for _, dep := range stage.DependsOn {
			downstreamMap[dep] = append(downstreamMap[dep], stage.StageID)
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Pipeline: %s\n", p.PipelineName))
	sb.WriteString(fmt.Sprintf("ID: %s\n", p.PipelineID))
	sb.WriteString(fmt.Sprintf("Stages: %d\n\n", len(p.Stages)))

	// Track which stages have been rendered
	rendered := make(map[string]bool)

	// Render in topological order
	for _, stageID := range order {
		stage := p.GetStage(stageID)
		if stage == nil {
			continue
		}

		// Render the stage
		sb.WriteString(renderStageNode(stage))

		// Mark as rendered
		rendered[stageID] = true

		// Render connections to downstream stages
		downstream := downstreamMap[stageID]
		if len(downstream) > 0 {
			sb.WriteString(renderConnections(stage, downstream))
		}

		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// renderStageNode renders a single stage node
func renderStageNode(stage *Stage) string {
	var sb strings.Builder

	// Stage header
	sb.WriteString(fmt.Sprintf("┌─ %s\n", stage.StageID))

	// Instance info
	instanceInfo := stage.InstanceType
	if stage.InstanceCount > 1 {
		instanceInfo += fmt.Sprintf(" (×%d)", stage.InstanceCount)
	}
	sb.WriteString(fmt.Sprintf("│  Type: %s\n", instanceInfo))

	// Region
	if stage.Region != "" {
		sb.WriteString(fmt.Sprintf("│  Region: %s\n", stage.Region))
	}

	// Spot
	if stage.Spot {
		sb.WriteString("│  Spot: true\n")
	}

	// EFA
	if stage.EFAEnabled {
		sb.WriteString("│  EFA: enabled\n")
	}

	// Dependencies
	if len(stage.DependsOn) > 0 {
		sb.WriteString(fmt.Sprintf("│  Depends on: %s\n", strings.Join(stage.DependsOn, ", ")))
	}

	// Data mode
	if stage.DataInput != nil {
		sb.WriteString(fmt.Sprintf("│  Input: %s", stage.DataInput.Mode))
		if stage.DataInput.Mode == "stream" {
			sb.WriteString(fmt.Sprintf(" (%s)", stage.DataInput.Protocol))
		}
		sb.WriteString("\n")
	}
	if stage.DataOutput != nil {
		sb.WriteString(fmt.Sprintf("│  Output: %s", stage.DataOutput.Mode))
		if stage.DataOutput.Mode == "stream" {
			sb.WriteString(fmt.Sprintf(" (%s:%d)", stage.DataOutput.Protocol, stage.DataOutput.Port))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("└─")

	return sb.String()
}

// renderConnections renders arrows to downstream stages
func renderConnections(stage *Stage, downstream []string) string {
	if len(downstream) == 0 {
		return ""
	}

	var sb strings.Builder

	if len(downstream) == 1 {
		// Single downstream: simple arrow
		sb.WriteString("\n   │\n")
		sb.WriteString("   ▼\n")
	} else {
		// Multiple downstream: fan-out
		sb.WriteString("\n   │\n")
		for i, ds := range downstream {
			if i == 0 {
				sb.WriteString(fmt.Sprintf("   ├──▶ %s\n", ds))
			} else if i == len(downstream)-1 {
				sb.WriteString(fmt.Sprintf("   └──▶ %s\n", ds))
			} else {
				sb.WriteString(fmt.Sprintf("   ├──▶ %s\n", ds))
			}
		}
	}

	return sb.String()
}

// RenderSimpleGraph renders a simplified ASCII graph
func (p *Pipeline) RenderSimpleGraph() (string, error) {
	order, err := p.GetTopologicalOrder()
	if err != nil {
		return "", err
	}

	var sb strings.Builder

	for i, stageID := range order {
		stage := p.GetStage(stageID)
		if stage == nil {
			continue
		}

		// Indentation based on depth
		indent := ""
		if len(stage.DependsOn) > 0 {
			indent = "  "
		}

		// Render stage
		sb.WriteString(fmt.Sprintf("%s%s", indent, stage.StageID))

		// Show instance count if > 1
		if stage.InstanceCount > 1 {
			sb.WriteString(fmt.Sprintf(" (×%d)", stage.InstanceCount))
		}

		sb.WriteString("\n")

		// Arrow to next if not last
		if i < len(order)-1 {
			nextStage := p.GetStage(order[i+1])
			if nextStage != nil {
				// Check if next stage depends on this stage
				dependsOnThis := false
				for _, dep := range nextStage.DependsOn {
					if dep == stageID {
						dependsOnThis = true
						break
					}
				}

				if dependsOnThis {
					sb.WriteString("   │\n")
					sb.WriteString("   ▼\n")
				}
			}
		}
	}

	return sb.String(), nil
}

// GetGraphStats returns statistics about the pipeline graph
func (p *Pipeline) GetGraphStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Count stages by dependency level
	depthMap := make(map[int]int)
	for _, stage := range p.Stages {
		depth := len(stage.DependsOn)
		depthMap[depth]++
	}

	// Count total instances
	totalInstances := 0
	for _, stage := range p.Stages {
		totalInstances += stage.InstanceCount
	}

	// Find max fan-out
	maxFanOut := 0
	downstreamMap := make(map[string]int)
	for _, stage := range p.Stages {
		for _, dep := range stage.DependsOn {
			downstreamMap[dep]++
		}
	}
	for _, count := range downstreamMap {
		if count > maxFanOut {
			maxFanOut = count
		}
	}

	// Find max fan-in
	maxFanIn := 0
	for _, stage := range p.Stages {
		if len(stage.DependsOn) > maxFanIn {
			maxFanIn = len(stage.DependsOn)
		}
	}

	stats["total_stages"] = len(p.Stages)
	stats["total_instances"] = totalInstances
	stats["max_fan_out"] = maxFanOut
	stats["max_fan_in"] = maxFanIn
	stats["stages_by_depth"] = depthMap
	stats["has_streaming"] = p.HasStreamingStages()
	stats["has_efa"] = p.HasEFAStages()

	return stats
}
