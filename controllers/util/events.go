package util

import (
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
)

const (
	// EventTypeWarning is used to alert user of failures that require user action
	EventTypeWarning = "Warning"

	// EventReasonValidationFailed is used when the StorageCluster spec validation fails
	EventReasonValidationFailed = "FailedValidation"

	// EventReasonUninstallPending is used when the StorageCluster uninstall is Pending
	EventReasonUninstallPending = "UninstallPending"
)

// EventReporter is custom events reporter type which allows user to limit the events
type EventReporter struct {
	mux            sync.Mutex
	recorder       record.EventRecorder
	reportedEvents map[string]*eventObject

	// report events x times where x is count
	count int

	// report events after x minutes
	eventReportAfterMinutes int
}

// NewEventReporter returns EventReporter object
func NewEventReporter(recorder record.EventRecorder, maxCountInGivenTime, reportAfter int) *EventReporter {
	er := &EventReporter{
		recorder:                recorder,
		count:                   maxCountInGivenTime,
		eventReportAfterMinutes: reportAfter,
	}

	er.reportedEvents = map[string]*eventObject{}

	return er
}

// Report records a events if eventReportAfterMinutes has passed or events occurred less than count
func (rep *EventReporter) Report(instance runtime.Object, eventType, eventReason, msg string) {
	rep.mux.Lock()
	defer rep.mux.Unlock()

	objMeta, err := meta.Accessor(instance)

	if err != nil {
		return
	}

	eventKey := fmt.Sprintf("%s:%s:%s:%s", objMeta.GetName(), eventType, eventReason, msg)

	eventobj, ok := rep.reportedEvents[eventKey]
	if !ok {
		obj := newEventObject()
		rep.reportedEvents[eventKey] = obj
		rep.recordEvent(instance, eventType, eventReason, msg)
	} else if eventobj.eventCount < rep.count {
		eventobj.eventCount++
		rep.recordEvent(instance, eventType, eventReason, msg)
	} else if eventobj.eventsReportedAt.Add(time.Minute * time.Duration(rep.eventReportAfterMinutes)).Before(time.Now()) {
		eventobj.eventCount = 1
		eventobj.eventsReportedAt = time.Now()
		rep.recordEvent(instance, eventType, eventReason, msg)
	}
}

func (rep *EventReporter) recordEvent(instance runtime.Object, eventType, eventReason, msg string) {
	rep.recorder.Event(instance, eventType, eventReason, msg)
}

type eventObject struct {
	eventsReportedAt time.Time
	eventCount       int
}

func newEventObject() *eventObject {
	return &eventObject{
		eventsReportedAt: time.Now(),
		eventCount:       1,
	}
}
