package agent

import (
	"context"
	"strings"
)

type PipelineStepFunc func(ctx context.Context, input string) (string, error)

func NewPipelineStepFromFunc(name string, fn PipelineStepFunc) PipelineStep {
	return &funcPipelineStep{name: name, fn: fn}
}

type funcPipelineStep struct {
	name string
	fn   PipelineStepFunc
}

func (s *funcPipelineStep) Name() string {
	return s.name
}

func (s *funcPipelineStep) Execute(ctx context.Context, input string) (string, error) {
	return s.fn(ctx, input)
}

type PipelineStep interface {
	Name() string
	Execute(ctx context.Context, input string) (string, error)
}

type PromptTemplateStep struct {
	template string
	vars     map[string]string
}

func NewPromptTemplateStep(template string, vars map[string]string) *PromptTemplateStep {
	return &PromptTemplateStep{template: template, vars: vars}
}

func (s *PromptTemplateStep) Name() string {
	return "prompt_template"
}

func (s *PromptTemplateStep) Execute(ctx context.Context, input string) (string, error) {
	result := s.template
	for k, v := range s.vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	result = strings.ReplaceAll(result, "{{input}}", input)
	return result, nil
}

type RouterStep struct {
	routes map[string]PipelineStep
}

func NewRouterStep(routes map[string]PipelineStep) *RouterStep {
	return &RouterStep{routes: routes}
}

func (s *RouterStep) Name() string {
	return "router"
}

func (s *RouterStep) Execute(ctx context.Context, input string) (string, error) {
	for route, step := range s.routes {
		if strings.Contains(strings.ToLower(input), strings.ToLower(route)) {
			return step.Execute(ctx, input)
		}
	}
	return input, nil
}
