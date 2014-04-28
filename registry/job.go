package registry

import (
	"errors"
	"path"
	"strings"

	etcdErr "github.com/coreos/fleet/third_party/github.com/coreos/etcd/error"
	"github.com/coreos/fleet/third_party/github.com/coreos/go-etcd/etcd"
	log "github.com/coreos/fleet/third_party/github.com/golang/glog"

	"github.com/coreos/fleet/event"
	"github.com/coreos/fleet/job"
	"github.com/coreos/fleet/unit"
)

const (
	jobPrefix = "/job/"
)

// GetAllJobs lists all Jobs known by the Registry
func (r *Registry) GetAllJobs() []job.Job {
	var jobs []job.Job

	key := path.Join(keyPrefix, jobPrefix)
	resp, err := r.etcd.Get(key, true, true)

	if err != nil {
		return jobs
	}

	for _, node := range resp.Node.Nodes {
		if j := r.GetJob(path.Base(node.Key)); j != nil {
			jobs = append(jobs, *j)
		}
	}

	return jobs
}

// GetAllJobsByMachine gets all Jobs assigned to the given machine
func (r *Registry) GetAllJobsByMachine(machBootID string) []job.Job {
	var jobs []job.Job

	key := path.Join(keyPrefix, jobPrefix)
	resp, err := r.etcd.Get(key, true, true)

	if err != nil {
		log.Errorf(err.Error())
		return jobs
	}

	for _, node := range resp.Node.Nodes {
		if j := r.GetJob(path.Base(node.Key)); j != nil {
			tgt := r.GetJobTarget(j.Name)
			if tgt != "" && tgt == machBootID {
				jobs = append(jobs, *j)
			}
		}
	}

	return jobs
}

// GetJobTarget looks up where the given job is scheduled. If the job has
// been scheduled, the boot ID the target machine is returned. Otherwise,
// an empty string is returned.
func (r *Registry) GetJobTarget(jobName string) string {
	// Figure out to which Machine this Job is scheduled
	key := jobTargetAgentPath(jobName)
	resp, err := r.etcd.Get(key, false, true)
	if err != nil {
		return ""
	}

	return resp.Node.Value
}

func (r *Registry) ClearJobTarget(jobName, bootID string) error {
	key := jobTargetAgentPath(jobName)
	_, err := r.etcd.CompareAndDelete(key, bootID, 0)
	return err
}

// GetJob looks for a Job of the given name in the Registry. It returns a fully
// hydrated Job on success, or nil on any kind of failure.
func (r *Registry) GetJob(jobName string) *job.Job {
	key := path.Join(keyPrefix, jobPrefix, jobName, "object")
	resp, err := r.etcd.Get(key, false, true)

	// Assume the error was KeyNotFound and return an empty data structure
	if err != nil {
		return nil
	}

	var jm jobModel
	if err := unmarshal(resp.Node.Value, &jm); err != nil {
		log.V(1).Infof("Error unmarshaling Job(%s): %v", jobName, err)
		return nil
	}

	var unit *unit.Unit

	// New-style Jobs should have a populated UnitHash, and the contents of the Unit are stored separately in the Registry
	if !jm.UnitHash.Empty() {
		unit = r.getUnitByHash(jm.UnitHash)
		if unit == nil {
			log.Warningf("No Unit found in Registry for Job(%s)", jm.Name)
			return nil
		}
		log.V(2).Infof("Got Unit for Job(%) from registry", jobName)
	} else {
		// Old-style Jobs had "Payloads" instead of Units, also stored separately in the Registry
		log.V(2).Infof("Legacy Job(%s) has no PayloadHash - looking for associated Payload", jm.Name)
		unit, err = r.getUnitFromLegacyPayload(jobName)
		if err != nil {
			log.Errorf("Error retrieving legacy payload for Job(%s)", jm.Name)
			return nil
		} else if unit == nil {
			log.Warningf("No Payload found in Registry for Job(%s)", jm.Name)
			return nil
		}
	}

	j := job.NewJob(jm.Name, *unit)

	j.UnitState = r.getUnitState(jm.Name)
	j.State = r.determineJobState(jm.Name)

	return j
}

// jobModel is used for deserializing jobs stored in the Registry
type jobModel struct {
	Name     string
	UnitHash unit.Hash
}

// DestroyJob removes a Job object from the repository, along with any legacy
// associated Payload. It also decrements the reference count on any associated
// Unit, and removes the Unit if this is the only Job referencing it. Finally, it removes any SignatureSets associated with the Job.
func (r *Registry) DestroyJob(jobName string) {
	key := path.Join(keyPrefix, jobPrefix, jobName)
	r.etcd.Delete(key, true)
	// TODO(jonboulle): add unit reference counting and actually destroying Units
	r.destroySignatureSetOfJob(jobName)
	r.destroyLegacyPayload(jobName)
	r.destroySignatureSetOfLegacyPayload(jobName)
}

// destroyLegacyPayload removes an old-style Payload from the registry
func (r *Registry) destroyLegacyPayload(payloadName string) {
	key := path.Join(keyPrefix, payloadPrefix, payloadName)
	r.etcd.Delete(key, false)
}

// CreateJob attempts to store a Job and its associated Unit in the registry
func (r *Registry) CreateJob(j *job.Job) (err error) {
	if err := r.storeOrGetUnit(j.Unit); err != nil {
		return err
	}

	key := path.Join(keyPrefix, jobPrefix, j.Name, "object")
	json, _ := marshal(j)

	_, err = r.etcd.Create(key, json, 0)
	if err != nil && err.(*etcd.EtcdError).ErrorCode == etcdErr.EcodeNodeExist {
		err = errors.New("job already exists")
	}

	return
}

func (r *Registry) GetJobTargetState(jobName string) *job.JobState {
	key := jobTargetStatePath(jobName)
	resp, err := r.etcd.Get(key, false, false)
	if err != nil {
		if err.(*etcd.EtcdError).ErrorCode != etcdErr.EcodeNodeExist {
			log.Errorf("Unable to determine target-state of Job(%s): %v", jobName, err)
		}
		return nil
	}

	return job.ParseJobState(resp.Node.Value)
}

func (r *Registry) SetJobTargetState(jobName string, state job.JobState) error {
	key := jobTargetStatePath(jobName)
	_, err := r.etcd.Set(key, string(state), 0)
	return err
}

func (es *EventStream) filterJobTargetStateChanges(resp *etcd.Response) *event.Event {
	if resp.Action != "set" {
		return nil
	}

	dir, baseName := path.Split(resp.Node.Key)
	if baseName != "target-state" {
		return nil
	}

	jobName := path.Base(strings.TrimSuffix(dir, "/"))

	ts := job.ParseJobState(resp.Node.Value)
	if ts == nil {
		return nil
	}

	cs := es.registry.determineJobState(jobName)
	if *cs == *ts {
		return nil
	}

	var cType string
	switch *cs {
	case job.JobStateInactive:
		cType = "CommandLoadJob"
	case job.JobStateLoaded:
		if *ts == job.JobStateInactive {
			cType = "CommandUnloadJob"
		} else if *ts == job.JobStateLaunched {
			cType = "CommandStartJob"
		}
	case job.JobStateLaunched:
		if *ts == job.JobStateLoaded {
			cType = "CommandStopJob"
		} else if *ts == job.JobStateInactive {
			cType = "CommandUnloadJob"
		}
	}

	if cType == "" {
		return nil
	}

	agent := es.registry.GetJobTarget(jobName)
	return &event.Event{cType, jobName, agent}
}

func (r *Registry) ScheduleJob(jobName string, machBootID string) error {
	key := jobTargetAgentPath(jobName)
	_, err := r.etcd.Create(key, machBootID, 0)
	return err
}

func (r *Registry) LockJob(jobName, context string) *TimedResourceMutex {
	return r.lockResource("job", jobName, context)
}

func filterEventJobScheduled(resp *etcd.Response) *event.Event {
	if resp.Action != "create" {
		return nil
	}

	dir, baseName := path.Split(resp.Node.Key)
	if baseName != "target" {
		return nil
	}

	jobName := path.Base(strings.TrimSuffix(dir, "/"))

	return &event.Event{"EventJobScheduled", jobName, resp.Node.Value}
}

func filterEventJobUnscheduled(resp *etcd.Response) *event.Event {
	if resp.Action != "delete" && resp.Action != "compareAndDelete" {
		return nil
	}

	dir, baseName := path.Split(resp.Node.Key)
	if baseName != "target" {
		return nil
	}

	if resp.PrevNode == nil {
		return nil
	}

	jobName := path.Base(strings.TrimSuffix(dir, "/"))
	return &event.Event{"EventJobUnscheduled", jobName, resp.PrevNode.Value}
}

func filterEventJobDestroyed(resp *etcd.Response) *event.Event {
	if resp.Action != "delete" {
		return nil
	}

	dir, jobName := path.Split(resp.Node.Key)
	dir = strings.TrimSuffix(dir, "/")
	dir, prefixName := path.Split(dir)

	if prefixName != "job" {
		return nil
	}

	return &event.Event{"EventJobDestroyed", jobName, nil}
}

func jobTargetAgentPath(jobName string) string {
	return path.Join(keyPrefix, jobPrefix, jobName, "target")
}

func jobTargetStatePath(jobName string) string {
	return path.Join(keyPrefix, jobPrefix, jobName, "target-state")
}
