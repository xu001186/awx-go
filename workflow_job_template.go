package awx

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// JobStatus the job status
type JobStatus string

const (
	PENDING    = JobStatus("pending")
	WAITING    = JobStatus("waiting")
	RUNNING    = JobStatus("running")
	SUCCESSFUL = JobStatus("successful")
	FAILED     = JobStatus("failed")
	ERROR      = JobStatus("error")
	CANCELED   = JobStatus("canceled")
)

// WorkflowJobTemplateService implements awx job template apis.
type WorkflowJobTemplateService struct {
	client *Client
}

// ListWorkflowJobTemplatesResponse represents `ListJobTemplates` endpoint response.
type ListWorkflowJobTemplatesResponse struct {
	Pagination
	Results []*WorkflowJobTemplate `json:"results"`
}

// ListWorkflowJobTemplates shows a list of job templates.
func (jt *WorkflowJobTemplateService) ListWorkflowJobTemplates(params map[string]string) ([]*WorkflowJobTemplate, *ListWorkflowJobTemplatesResponse, error) {
	result := new(ListWorkflowJobTemplatesResponse)
	endpoint := "/api/v2/workflow_job_templates/"
	resp, err := jt.client.Requester.GetJSON(endpoint, result, params)
	if err != nil {
		return nil, result, err
	}

	if err := CheckResponse(resp); err != nil {
		return nil, result, err
	}

	return result.Results, result, nil
}

// GetWorkflowJobTemplateByName Get the workflowJob details by its name
func (jt *WorkflowJobTemplateService) GetWorkflowJobTemplateByName(name string) (*WorkflowJobTemplate, error) {
	params := make(map[string]string)
	params["name"] = name
	result := new(ListWorkflowJobTemplatesResponse)
	endpoint := "/api/v2/workflow_job_templates/"
	resp, err := jt.client.Requester.GetJSON(endpoint, result, params)
	if err != nil {
		return nil, err
	}

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	if result.Count == 0 {
		return nil, fmt.Errorf("The %v can't be found")
	}

	return result.Results[0], nil
}

// Launch Workflow doesn't support the extra variables
func (jt *WorkflowJobTemplateService) Launch(id int, jobHanlder JobHanlder, jobCheckInterval, jobCheckmaxtries int) (*WorkflowJobLaunch, *Job, error) {

	result := new(WorkflowJobLaunch)
	endpoint := fmt.Sprintf("/api/v2/workflow_job_templates/%d/launch/", id)
	resp, err := jt.client.Requester.PostJSON(endpoint, nil, result, nil)
	if err != nil {
		return nil, nil, err
	}

	if err := CheckResponse(resp); err != nil {
		return nil, nil, err
	}

	// in case invalid job id return
	if result.WorkflowJob == 0 {
		return nil, nil, errors.New("invalid job id 0")
	}
	jobresult, err := jobHanlder(jt, result.WorkflowJob, jobCheckInterval, jobCheckmaxtries)
	return result, jobresult, err
}

//JobHanlder handler the job
type JobHanlder func(jt *WorkflowJobTemplateService, jobid, checkInterval, maxtries int) (*Job, error)

//CheckOnce check the result once
func CheckOnce(jt *WorkflowJobTemplateService, jobid, checkInterval, maxtries int) (*Job, error) {
	result := &Job{}
	endpoint := fmt.Sprintf("/api/v2/workflow_jobs/%d", jobid)
	resp, err := jt.client.Requester.GetJSON(endpoint, result, nil)
	if err != nil {
		return result, err
	}

	if err := CheckResponse(resp); err != nil {
		return result, err
	}
	return result, err
}

//DefaultJobHandler wait the job util it's not PENDING,WAITING or RUNNING
func DefaultJobHandler(jt *WorkflowJobTemplateService, jobid, checkInterval, maxtries int) (*Job, error) {
	result := &Job{}
	quitChan := make(chan error)
	tryCount := 0
	go func(quitChan chan error, tryCount int) {
		for {
			endpoint := fmt.Sprintf("/api/v2/workflow_jobs/%d", jobid)
			resp, err := jt.client.Requester.GetJSON(endpoint, result, nil)
			if err != nil {
				quitChan <- err
			}

			if err := CheckResponse(resp); err != nil {
				quitChan <- err
				return
			}
			if result.Status != string(PENDING) && result.Status != string(WAITING) && result.Status != string(RUNNING) {
				quitChan <- nil
				return
			}
			tryCount++
			if tryCount <= maxtries && maxtries != -1 {
				quitChan <- fmt.Errorf("The maximum number %v of checking job status has been reached ", maxtries)
				return
			}
			time.Sleep(time.Duration(checkInterval) * time.Second)
		}
	}(quitChan, tryCount)
	err := <-quitChan
	return result, err
}

// CreateJobTemplate creates a job template
func (jt *WorkflowJobTemplateService) CreateJobTemplate(data map[string]interface{}, params map[string]string) (*JobTemplate, error) {
	result := new(JobTemplate)
	mandatoryFields = []string{"name", "job_type", "inventory", "project"}
	validate, status := ValidateParams(data, mandatoryFields)
	if !status {
		err := fmt.Errorf("Mandatory input arguments are absent: %s", validate)
		return nil, err
	}
	endpoint := "/api/v2/workflow_job_templates/"
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	resp, err := jt.client.Requester.PostJSON(endpoint, bytes.NewReader(payload), result, params)
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateJobTemplate updates a job template
func (jt *WorkflowJobTemplateService) UpdateJobTemplate(id int, data map[string]interface{}, params map[string]string) (*JobTemplate, error) {
	result := new(JobTemplate)
	endpoint := fmt.Sprintf("/api/v2/workflow_job_templates/%d", id)
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	resp, err := jt.client.Requester.PatchJSON(endpoint, bytes.NewReader(payload), result, params)
	if err != nil {
		return nil, err
	}
	if err := CheckResponse(resp); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteJobTemplate deletes a job template
func (jt *WorkflowJobTemplateService) DeleteJobTemplate(id int) (*JobTemplate, error) {
	result := new(JobTemplate)
	endpoint := fmt.Sprintf("/api/v2/workflow_job_templates/%d", id)

	resp, err := jt.client.Requester.Delete(endpoint, result, nil)
	if err != nil {
		return nil, err
	}

	if err := CheckResponse(resp); err != nil {
		return nil, err
	}

	return result, nil
}
