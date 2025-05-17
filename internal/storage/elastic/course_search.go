package elastic

import (
	"SkillForge/internal/models"
	custom_json "SkillForge/pkg/custom_serializer/json"
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
	return &CourseSearchRepo{client: client, index: index}
}

func (r *CourseSearchRepo) CreateIndexIfNotExist(ctx context.Context) error {
	s := custom_json.New()
	existsReq := esapi.IndicesExistsRequest{Index: []string{r.index}}
	existsRes, err := existsReq.Do(ctx, r.client)
	if err != nil {
		return fmt.Errorf("error checking index existence: %w", err)
	}
	defer existsRes.Body.Close()

	if existsRes.StatusCode == 404 {
		mapping := map[string]interface{}{
			"settings": map[string]interface{}{
				"analysis": map[string]interface{}{
					"analyzer": map[string]interface{}{
						"edge_ngram_analyzer": map[string]interface{}{
							"tokenizer": "edge_ngram_tokenizer",
							"filter":    []string{"lowercase"},
						},
					},
					"tokenizer": map[string]interface{}{
						"edge_ngram_tokenizer": map[string]interface{}{
							"type":        "edge_ngram",
							"min_gram":    2,
							"max_gram":    20,
							"token_chars": []string{"letter", "digit"},
						},
					},
				},
			},
			"mappings": map[string]interface{}{
				"properties": map[string]interface{}{
					"title": map[string]interface{}{
						"type":            "text",
						"analyzer":        "edge_ngram_analyzer",
						"search_analyzer": "standard",
					},
					"description": map[string]interface{}{
						"type":            "text",
						"analyzer":        "edge_ngram_analyzer",
						"search_analyzer": "standard",
					},
				},
			},
		}

		body, _ := s.Marshal(mapping)
		req := esapi.IndicesCreateRequest{Index: r.index, Body: bytes.NewReader(body)}
		res, err := req.Do(ctx, r.client)
		if err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
		defer res.Body.Close()
		if res.IsError() {
			return fmt.Errorf("mapping creation failed: %s", res.String())
		}
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

func (r *CourseSearchRepo) Count(ctx context.Context, query string) (int, error) {
	q := map[string]any{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":                query,
				"fields":               []string{"title^3", "description"},
				"type":                 "best_fields",
				"fuzziness":            "AUTO",
				"operator":             "or",
				"minimum_should_match": "2<75%",
			},
		},
	}
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(q); err != nil {
		return 0, fmt.Errorf("encode count body: %w", err)
	}
	req := esapi.CountRequest{Index: []string{r.index}, Body: buf}
	res, err := req.Do(ctx, r.client)
	if err != nil {
		return 0, fmt.Errorf("count request failed: %w", err)
	}
	defer res.Body.Close()
	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return 0, fmt.Errorf("count error: %s", string(bodyBytes))
	}
	var cntRes struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(res.Body).Decode(&cntRes); err != nil {
		return 0, fmt.Errorf("decode count response: %w", err)
	}
	return cntRes.Count, nil
}

func (r *CourseSearchRepo) Search(ctx context.Context, query string, size int) ([]uuid.UUID, error) {
	if size <= 0 {
		size = 10
	}
	q := map[string]interface{}{
		"query": map[string]interface{}{
			"multi_match": map[string]interface{}{
				"query":                query,
				"fields":               []string{"title^3", "description"},
				"type":                 "best_fields",
				"fuzziness":            "AUTO",
				"operator":             "or",
				"minimum_should_match": "2<75%",
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
	if res.IsError() {
		bodyBytes, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("search error: %s", string(bodyBytes))
	}
	var esRes struct {
		Hits struct {
			Hits []struct {
				ID string `json:"_id"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&esRes); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	var ids []uuid.UUID
	for _, h := range esRes.Hits.Hits {
		if id, err := uuid.Parse(h.ID); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
