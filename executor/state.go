// Copyright (c) 2018 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package executor

import (
	"context"
	"fmt"

	"github.com/m3db/m3coordinator/executor/transform"
	"github.com/m3db/m3coordinator/parser"
	"github.com/m3db/m3coordinator/plan"
	"github.com/m3db/m3coordinator/storage"
	"github.com/m3db/m3coordinator/util/execution"

	"github.com/pkg/errors"
)

// ExecutionState represents the execution hierarchy
type ExecutionState struct {
	plan       plan.PhysicalPlan
	sources    []parser.Source
	resultNode Result
	storage    storage.Storage
}

// CreateSource creates a source node
func CreateSource(ID parser.NodeID, params SourceParams, storage storage.Storage, options transform.Options) (parser.Source, *transform.Controller) {
	controller := &transform.Controller{ID: ID}
	return params.Node(controller, storage, options), controller
}

// CreateTransform creates a transform node which works on functions and contains state
func CreateTransform(ID parser.NodeID, params TransformParams) (transform.OpNode, *transform.Controller) {
	controller := &transform.Controller{ID: ID}
	return params.Node(controller), controller
}

// TransformParams are defined by transforms
type TransformParams interface {
	parser.Params
	Node(controller *transform.Controller) transform.OpNode
}

// SourceParams are defined by sources
type SourceParams interface {
	parser.Params
	Node(controller *transform.Controller, storage storage.Storage, options transform.Options) parser.Source
}

// GenerateExecutionState creates an execution state from the physical plan
func GenerateExecutionState(pplan plan.PhysicalPlan, storage storage.Storage) (*ExecutionState, error) {
	result := pplan.ResultStep
	state := &ExecutionState{
		plan:    pplan,
		storage: storage,
	}

	step, ok := pplan.Step(result.Parent)
	if !ok {
		return nil, fmt.Errorf("incorrect parent reference in result node, parentId: %s", result.Parent)
	}

	options := transform.Options{
		Now: pplan.Now,
	}
	controller, err := state.createNode(step, options)
	if err != nil {
		return nil, err
	}

	if len(state.sources) == 0 {
		return nil, errors.New("empty sources for the execution state")
	}

	rNode := ResultNode{}
	state.resultNode = rNode
	controller.AddTransform(rNode)

	return state, nil
}

// createNode helps to create an execution node recursively
// TODO: consider modifying this function so that ExecutionState can have a non pointer receiver
func (s *ExecutionState) createNode(step plan.LogicalStep, options transform.Options) (*transform.Controller, error) {
	// TODO: consider using a registry instead of casting to an interface
	sourceParams, ok := step.Transform.Op.(SourceParams)
	if ok {
		source, controller := CreateSource(step.ID(), sourceParams, s.storage, options)
		s.sources = append(s.sources, source)
		return controller, nil
	}

	transformParams, ok := step.Transform.Op.(TransformParams)
	if !ok {
		return nil, fmt.Errorf("invalid transform step, %s", step)
	}

	transformNode, controller := CreateTransform(step.ID(), transformParams)
	for _, parentID := range step.Parents {
		parentStep, ok := s.plan.Step(parentID)
		if !ok {
			return nil, fmt.Errorf("incorrect parent reference, parentId: %s, node: %s", parentID, step.ID())
		}

		parentController, err := s.createNode(parentStep, options)
		if err != nil {
			return nil, err
		}

		parentController.AddTransform(transformNode)
	}

	return controller, nil
}

// Execute the sources in parallel and return the first error
func (s *ExecutionState) Execute(ctx context.Context) error {
	requests := make([]execution.Request, len(s.sources))
	for idx, source := range s.sources {
		requests[idx] = sourceRequest{source}
	}

	return execution.ExecuteParallel(ctx, requests)
}

// String representation of the state
func (s *ExecutionState) String() string {
	return fmt.Sprintf("plan: %s\nsources: %s\nresult: %s", s.plan, s.sources, s.resultNode)
}

type sourceRequest struct {
	source parser.Source
}

func (s sourceRequest) Process(ctx context.Context) error {
	return s.source.Execute(ctx)
}
