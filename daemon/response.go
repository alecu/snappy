// -*- Mode: Go; indent-tabs-mode: t -*-

/*
 * Copyright (C) 2015 Canonical Ltd
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License version 3 as
 * published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package daemon

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/ubuntu-core/snappy/asserts"
	"github.com/ubuntu-core/snappy/logger"
)

// ResponseType is the response type
type ResponseType string

// “there are three standard return types: Standard return value,
// Background operation, Error”, each returning a JSON object with the
// following “type” field:
const (
	ResponseTypeSync  ResponseType = "sync"
	ResponseTypeAsync ResponseType = "async"
	ResponseTypeError ResponseType = "error"
)

// Response knows how to serve itself, and how to find itself
type Response interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	Self(*Command, *http.Request) Response // has the same arity as ResponseFunc for convenience
}

type resp struct {
	Type   ResponseType `json:"type"`
	Status int          `json:"status_code"`
	Result interface{}  `json:"result"`
}

func (r *resp) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":        r.Type,
		"status":      http.StatusText(r.Status),
		"status_code": r.Status,
		"result":      &r.Result,
	})
}

func (r *resp) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	status := r.Status
	bs, err := r.MarshalJSON()
	if err != nil {
		logger.Noticef("unable to marshal %#v to JSON: %v", *r, err)
		bs = nil
		status = http.StatusInternalServerError
	}

	hdr := w.Header()
	if r.Status == http.StatusAccepted || r.Status == http.StatusCreated {
		if m, ok := r.Result.(map[string]interface{}); ok {
			if location, ok := m["resource"]; ok {
				if location, ok := location.(string); ok && location != "" {
					hdr.Set("Location", location)
				}
			}
		}
	}

	hdr.Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(bs)
}

func (r *resp) Self(*Command, *http.Request) Response {
	return r
}

type errorKind string

const (
	errorKindLicenseRequired = errorKind("license-required")
)

type errorValue interface{}

type errorResult struct {
	Message string     `json:"message"` // note no omitempty
	Kind    errorKind  `json:"kind,omitempty"`
	Value   errorValue `json:"value,omitempty"`
}

// SyncResponse builds a "sync" response from the given result.
func SyncResponse(result interface{}) Response {
	if err, ok := result.(error); ok {
		return InternalError("internal error: %v", err)
	}

	if rsp, ok := result.(Response); ok {
		return rsp
	}

	return &resp{
		Type:   ResponseTypeSync,
		Status: http.StatusOK,
		Result: result,
	}
}

// AsyncResponse builds an "async" response from the given *Task
func AsyncResponse(result map[string]interface{}) Response {
	return &resp{
		Type:   ResponseTypeAsync,
		Status: http.StatusAccepted,
		Result: result,
	}
}

// makeErrorResponder builds an errorResponder from the given error status.
func makeErrorResponder(status int) errorResponder {
	return func(format string, v ...interface{}) Response {
		return &resp{
			Type: ResponseTypeError,
			Result: &errorResult{
				Message: fmt.Sprintf(format, v...),
			},
			Status: status,
		}
	}
}

// A FileResponse 's ServeHTTP method serves the file
type FileResponse string

// Self from the Response interface
func (f FileResponse) Self(*Command, *http.Request) Response { return f }

// ServeHTTP from the Response interface
func (f FileResponse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	filename := fmt.Sprintf("attachment; filename=%s", filepath.Base(string(f)))
	w.Header().Add("Content-Disposition", filename)
	http.ServeFile(w, r, string(f))
}

type assertResponse struct {
	assertions []asserts.Assertion
	bundle     bool
}

// AssertResponse builds a response whose ServerHTTP method serves one or a bundle of assertions.
func AssertResponse(asserts []asserts.Assertion, bundle bool) Response {
	if len(asserts) > 1 {
		bundle = true
	}
	return &assertResponse{assertions: asserts, bundle: bundle}
}

func (ar assertResponse) Self(*Command, *http.Request) Response { return ar }

func (ar assertResponse) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t := asserts.MediaType
	if ar.bundle {
		t = mime.FormatMediaType(t, map[string]string{"bundle": "y"})
	}
	w.Header().Set("Content-Type", t)
	w.Header().Set("X-Ubuntu-Assertions-Count", strconv.Itoa(len(ar.assertions)))
	enc := asserts.NewEncoder(w)
	for _, a := range ar.assertions {
		err := enc.Encode(a)
		if err != nil {
			logger.Noticef("unable to write encoded assertion into response: %v", err)
			break

		}
	}
}

// errorResponder is a callable that produces an error Response.
// e.g., InternalError("something broke: %v", err), etc.
type errorResponder func(string, ...interface{}) Response

// standard error responses
var (
	NotFound       = makeErrorResponder(http.StatusNotFound)
	BadRequest     = makeErrorResponder(http.StatusBadRequest)
	BadMethod      = makeErrorResponder(http.StatusMethodNotAllowed)
	InternalError  = makeErrorResponder(http.StatusInternalServerError)
	NotImplemented = makeErrorResponder(http.StatusNotImplemented)
	Forbidden      = makeErrorResponder(http.StatusForbidden)
)
