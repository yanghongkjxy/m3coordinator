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

package ts

import (
	"context"
	"testing"
	"time"

	"github.com/m3db/m3coordinator/models"

	"github.com/stretchr/testify/assert"
)

func TestCreateNewSeries(t *testing.T) {
	ctx := context.TODO()
	startTime := time.Now()
	tags := models.Tags{"foo": "bar", "biz": "baz"}
	values := newValues(ctx, 1000, 10000, 1)
	series := NewSeries(ctx, "metrics", startTime, values, tags)

	assert.Equal(t, "metrics", series.Name())
	assert.Equal(t, 10000, series.Len())
	assert.Equal(t, 1000, series.MillisPerStep())
	assert.Equal(t, 1.0, series.ValueAt(0))
	assert.Equal(t, startTime, series.StartTime())
	assert.Equal(t, startTime, series.StartTimeForStep(0))
}
