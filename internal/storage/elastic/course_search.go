package elastic

import (
	"SkillForge/internal/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/google/uuid"
	"io"
)

type CourseSearchRepo struct {
	client *elasticsearch.Client
	index  string
}

func NewCourseSearchRepository(client *elasticsearch.Client, index string) *CourseSearchRepo {
	return &CourseSearchRepo{
		client: client,
		index:  index,
	}
}

func (r *CourseSearchRepo) CreateIndexIfNotExist(ctx context.Context) error {
	existsReq := esapi.IndicesExistsRequest{Index: []string{r.index}}
	existsRes, err := existsReq.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("error checking index existence: %w", err)
	}
	defer existsRes.Body.Close()

	if existsRes.StatusCode == 404 {
		mapping := map[string]interface{}{
			"mappings": map[string]interface{}{"properties": map[string]interface{}{
				"title":       map[string]string{"type": "text"},
				"description": map[string]string{"type": "text"},
			}},
		}
		body, _ := json.Marshal(mapping)
		req := esapi.IndicesCreateRequest{Index: r.index, Body: bytes.NewReader(body)}
		res, err := req.Do(ctx, r.client)
		if err != nil {
			return err
		}
		defer res.Body.Close()
	}
	if existsRes.StatusCode >= 300 && existsRes.StatusCode != 404 {
		return fmt.Errorf("index existence check failed with status code %d", existsRes.StatusCode)
	}
	return nil
}

func (r *CourseSearchRepo) Index(ctx context.Context, course models.Course) error {
	doc := map[string]interface{}{
		"title":       course.Title,
		"description": course.Description,
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal doc: %w", err)
	}

	req := esapi.IndexRequest{
		Index:      r.index,
		DocumentID: course.ID.String(),
		Refresh:    "true",
		Body:       bytes.NewReader(data),
	}
	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("index request: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("index error: %s", res.String())
	}
	return nil
}

func (r *CourseSearchRepo) Update(ctx context.Context, course models.Course) error {
	partial := map[string]interface{}{"doc": map[string]interface{}{
		"title":       course.Title,
		"description": course.Description,
	}}
	body, err := json.Marshal(partial)
	if err != nil {
		return fmt.Errorf("marshal update: %w", err)
	}

	req := esapi.UpdateRequest{
		Index:      r.index,
		DocumentID: course.ID.String(),
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}
	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("update request: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("update error: %s", res.String())
	}
	return nil
}

func (r *CourseSearchRepo) Delete(ctx context.Context, id uuid.UUID) error {
	req := esapi.DeleteRequest{
		Index:      r.index,
		DocumentID: id.String(),
		Refresh:    "true",
	}
	res, err := req.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("delete request: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		return fmt.Errorf("delete error: %s", res.String())
	}
	return nil
}

func (r *CourseSearchRepo) Search(ctx context.Context, query string, size int) ([]uuid.UUID, error) {
	if size <= 0 {
		size = 10
	}

	q := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":     query,
				"fields":    []string{"title", "description"},
				"type":      "best_fields",
				"fuzziness": "AUTO",
			},
		},
		"size": size,
	}
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(q); err != nil {
		return nil, fmt.Errorf("encode search body: %w", err)
	}

	res, err := r.client.Search(
		r.client.Search.WithContext(ctx),
		r.client.Search.WithIndex(r.index),
		r.client.Search.WithBody(buf),
	)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("search error: status %s, body: %s", res.Status(), string(bodyBytes))
	}

	var esRes struct {
		Hits struct {
			Hits []struct {
				ID string `json:"_id"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err = json.NewDecoder(res.Body).Decode(&esRes); err != nil {
		return nil, err
	}

	var ids []uuid.UUID
	for _, h := range esRes.Hits.Hits {
		id, err := uuid.Parse(h.ID)
		if err != nil {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}
