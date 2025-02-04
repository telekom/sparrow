// SPDX-FileCopyrightText: 2025 Deutsche Telekom IT GmbH
//
// SPDX-License-Identifier: Apache-2.0

package db

import (
	"reflect"
	"sync"
	"testing"

	"github.com/telekom/sparrow/pkg/checks"
)

func TestInMemory_Save(t *testing.T) {
	type fields struct {
		data map[string]checks.Result
	}
	type args struct {
		result checks.ResultDTO
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{name: "Saves without error", fields: fields{data: make(map[string]checks.Result)}, args: args{result: checks.ResultDTO{Name: "Test", Result: &checks.Result{Data: 0}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &InMemory{
				data: sync.Map{},
			}
			for k, v := range tt.fields.data {
				i.data.Store(k, v)
			}

			i.Save(tt.args.result)
			val, ok := i.data.Load(tt.args.result.Name)
			if !ok {
				t.Fatalf("Expected to find key %s in map", tt.args.result.Name)
			}

			if !reflect.DeepEqual(val, tt.args.result.Result) {
				t.Fatalf("Expected val to be %v but got: %v", val, tt.args.result.Result)
			}
		})
	}
}

func TestNewInMemory(t *testing.T) {
	tests := []struct {
		name string
		want *InMemory
	}{
		{name: "Creates without nil pointers", want: &InMemory{data: sync.Map{}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewInMemory(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewInMemory() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInMemory_Get(t *testing.T) {
	type fields struct {
		data map[string]*checks.Result
	}
	type args struct {
		check string
	}
	type want struct {
		check checks.Result
		ok    bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   want
	}{
		{name: "Can get value", fields: fields{data: map[string]*checks.Result{
			"alpha": {Data: 0},
			"beta":  {Data: 1},
		}}, want: want{ok: true, check: checks.Result{Data: 1}}, args: args{check: "beta"}},
		{name: "Not found", fields: fields{data: map[string]*checks.Result{
			"alpha": {Data: 0},
			"beta":  {Data: 1},
		}}, want: want{ok: false, check: checks.Result{}}, args: args{check: "NOTFOUND"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &InMemory{
				data: sync.Map{},
			}
			for k, v := range tt.fields.data {
				i.data.Store(k, v)
			}
			if got, ok := i.Get(tt.args.check); !reflect.DeepEqual(got, tt.want.check) || ok != tt.want.ok {
				t.Errorf("Get() = %v, want %v", got, tt.want.check)
				t.Errorf("Ok = %v, want %v", ok, tt.want.ok)
			}
		})
	}
}

func TestInMemory_List(t *testing.T) {
	type fields struct {
		data map[string]*checks.Result
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]checks.Result
	}{
		{name: "Lists all entries", fields: fields{
			data: map[string]*checks.Result{
				"alpha": {Data: 0},
				"beta":  {Data: 1},
			},
		}, want: map[string]checks.Result{
			"alpha": {Data: 0},
			"beta":  {Data: 1},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &InMemory{
				data: sync.Map{},
			}
			for k, v := range tt.fields.data {
				i.data.Store(k, v)
			}

			got := i.List()

			if got == nil {
				t.Fatalf("Expected got != nil")
			}

			if !reflect.DeepEqual(tt.want, got) {
				defer t.Fail()
				t.Log("tt.want != got")
				for k, v := range tt.want {
					found, ok := i.data.Load(k)
					if !ok {
						t.Logf("Failed to find expected key %s", k)
					}
					if !reflect.DeepEqual(v, found) {
						t.Logf("Value for key %s in db does not equal %v got %v instead", k, v, found)
					}
				}
			}
		})
	}
}

func TestInMemory_ListThreadsafe(t *testing.T) {
	db := NewInMemory()
	db.Save(checks.ResultDTO{Name: "alpha", Result: &checks.Result{Data: 0}})
	db.Save(checks.ResultDTO{Name: "beta", Result: &checks.Result{Data: 1}})

	got := db.List()
	if len(got) != 2 {
		t.Errorf("Expected 2 entries but got %d", len(got))
	}

	got["alpha"] = checks.Result{Data: 50}

	newGot := db.List()
	if newGot["alpha"].Data != 0 {
		t.Errorf("Expected alpha to be 0 but got %d", newGot["alpha"].Data)
	}
}
