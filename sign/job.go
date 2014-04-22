package sign

import (
	"errors"
	"path"

	"github.com/coreos/fleet/job"
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
func (sv *SignatureVerifier) VerifyJob(j *job.Job, s *SignatureSet) (bool, error) {
	if s == nil {
		return false, errors.New("signature to verify is nil")
	}

	tag := TagForJob(j.Name)
	if tag != s.Tag {
		return false, errors.New("unmatched unit and signature")
	}

	data, _ := marshal(j.Unit)
	return sv.Verify(data, s)
}

/*

// SignPayload signs the provided JobPayload, returning a SignatureSet
func (sc *SignatureCreator) SignPayload(jp *job.JobPayload) (*SignatureSet, error) {
	tag := TagForPayload(payloadPrefix, jp.Name)
	data, _ := marshal(jp)
	return sc.Sign(tag, data)
}

// VerifyPayload verifies the payload using signature
func (sv *SignatureVerifier) VerifyPayload(jp *job.JobPayload, s *SignatureSet) (bool, error) {
	if s == nil {
		return false, errors.New("signature to verify is nil")
	}

	TagForPayload(payloadPrefix, jp.Name)
	if tag != s.Tag {
		return false, errors.New("unmatched payload and signature")
	}

	data, _ := marshal(jp)
	return sv.Verify(data, s)
}
*/
