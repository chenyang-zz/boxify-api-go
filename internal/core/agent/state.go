package agent

import (
	"context"

	"github.com/boxify/api-go/internal/core/llm"
	coretool "github.com/boxify/api-go/internal/core/tool"
)

func (a *Agent) transition(ctx context.Context, state *State, to Phase, reason string) error {
	transition := Transition{
		From:      state.Phase,
		To:        to,
		Iteration: state.Iteration,
		Reason:    reason,
	}
	if err := a.hooks.BeforeTransition(ctx, cloneState(*state), transition); err != nil {
		return err
	}
	state.Phase = to
	if err := a.hooks.AfterTransition(ctx, cloneState(*state), transition); err != nil {
		return err
	}
	return nil
}

func cloneState(state State) State {
	state.Input.Messages = cloneLLMMessages(state.Input.Messages)
	state.Tools = cloneToolDescriptors(state.Tools)
	state.Steps = cloneSteps(state.Steps)
	state.LastDecision.ActionInput = cloneInput(state.LastDecision.ActionInput)
	return state
}

func cloneLLMMessages(messages []*llm.Message) []*llm.Message {
	if messages == nil {
		return nil
	}
	out := make([]*llm.Message, 0, len(messages))
	for _, message := range messages {
		if message == nil {
			out = append(out, nil)
			continue
		}
		copied := *message
		out = append(out, &copied)
	}
	return out
}

func cloneToolDescriptors(descriptors []coretool.Descriptor) []coretool.Descriptor {
	if descriptors == nil {
		return nil
	}
	out := make([]coretool.Descriptor, len(descriptors))
	copy(out, descriptors)
	for i := range out {
		out[i].Schema.Parameters.Properties = cloneProperties(out[i].Schema.Parameters.Properties)
		out[i].Schema.Parameters.Required = cloneStrings(out[i].Schema.Parameters.Required)
		out[i].Annotations = cloneAnyMap(out[i].Annotations)
	}
	return out
}

func cloneSteps(steps []Step) []Step {
	if steps == nil {
		return nil
	}
	out := make([]Step, len(steps))
	copy(out, steps)
	for i := range out {
		out[i].ActionInput = cloneInput(out[i].ActionInput)
	}
	return out
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

func cloneProperties(properties map[string]coretool.PropertySchema) map[string]coretool.PropertySchema {
	if properties == nil {
		return nil
	}
	out := make(map[string]coretool.PropertySchema, len(properties))
	for key, value := range properties {
		out[key] = coretool.PropertySchema(cloneAnyMap(value))
	}
	return out
}

func cloneAnyMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}
