package react

import (
	"github.com/boxify/api-go/internal/core/llm"
	coretool "github.com/boxify/api-go/internal/core/tool"
)

func cloneState(state State) State {
	state.Input.Messages = llm.CloneMessages(state.Input.Messages)
	state.Tools = cloneToolDescriptors(state.Tools)
	state.Steps = cloneSteps(state.Steps)
	state.LastDecision = cloneDecision(state.LastDecision)
	return state
}

func cloneDecision(decision Decision) Decision {
	decision.ActionInput = cloneInput(decision.ActionInput)
	return decision
}

func cloneToolDescriptors(descriptors []coretool.Descriptor) []coretool.Descriptor {
	return coretool.CloneDescriptors(descriptors)
}

func cloneSteps(steps []Step) []Step {
	if steps == nil {
		return nil
	}
	out := make([]Step, len(steps))
	copy(out, steps)
	for i := range out {
		out[i] = cloneStep(out[i])
	}
	return out
}

func cloneStep(step Step) Step {
	step.ActionInput = cloneInput(step.ActionInput)
	return step
}

func cloneInput(input coretool.Input) coretool.Input {
	if input == nil {
		return nil
	}
	out := make(coretool.Input, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
