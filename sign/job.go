package sign

import (
	"errors"
	"path"

	"github.com/coreos/fleet/job"
	"github.com/coreos/fleet/unit"
)

const (
	unitPrefix = "/job"
	// Legacy JobPayloads had signatures stored in payloadPrefix
	payloadPrefix = "/payload/"
)

// TagForJob returns a tag used to identify and store signatures for a Job
func TagForJob(jobName string) string {
	return path.Join(unitPrefix, jobName)
}

// TagForPayload returns a tag use to store legacy JobPayload signatures
func TagForPayload(name string) string {
	return path.Join(payloadPrefix, name)
}

// SignJob signs the provided Job's Unit, returning a SignatureSet
func (sc *SignatureCreator) SignJob(j *job.Job) (*SignatureSet, error) {
	tag := TagForJob(j.Name)
	data, _ := marshal(j.Unit)
	return sc.Sign(tag, data)
}

// VerifyJob verifies the provided Job's Unit using the given SignatureSet
func (sv *SignatureVerifier) VerifyJob(j *job.Job, ss *SignatureSet) (bool, error) {
	if ss == nil {
		return false, errors.New("SignatureSet to verify is nil")
	}

	tag := TagForJob(j.Name)
	if tag != ss.Tag {
		return false, errors.New("unmatched unit and signature")
	}

	data, _ := marshal(j.Unit)
	return sv.Verify(data, ss)
}

// VerifyLegacyPayload verifies the payload using signature
func (sv *SignatureVerifier) VerifyLegacyPayload(jp *unit.LegacyJobPayload, s *SignatureSet) (bool, error) {
	if s == nil {
		return false, errors.New("signature to verify is nil")
	}

	tag := TagForPayload(jp.Name)
	if tag != s.Tag {
		return false, errors.New("unmatched payload and signature")
	}

	data, _ := marshal(jp)
	return sv.Verify(data, s)
}
