package influx

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/influxdata/chronograf"
)

const (
	// AllAnnotations returns all annotations from the chronograf database
	AllAnnotations = `SELECT "duration_ns", "text", "annoType", "id" FROM "chronograf"."autogen"."annotations" WHERE "deleted"='false' ORDER BY time DESC`
	// GetAnnotationID returns all annotations from the chronograf database where id is %s
	GetAnnotationID = `SELECT "duration_ns", "text", "annoType", "id" FROM "chronograf"."autogen"."annotations" WHERE "id"='%s' AND "deleted"='false' ORDER BY time DESC`
)

// AnnotationStore stores annotations within InfluxDB
type AnnotationStore struct {
	client Client
}

// All lists all Annotations
func (a *AnnotationStore) All(ctx context.Context) ([]chronograf.Annotation, error) {
	return a.client.allAnnotations(ctx)
}

// Get retrieves an annotation
func (a *AnnotationStore) Get(ctx context.Context, id string) (chronograf.Annotation, error) {
	annos, err := a.client.getAnnotations(ctx, id)
	if err != nil {
		return chronograf.Annotation{}, err
	}
	if len(annos) == 0 {
		// TODO: change this to a chronograf.Error
		return chronograf.Annotation{}, fmt.Errorf("Unknown annotation id %s", id)
	}
	return annos[0], nil
}

// Add creates a new annotation in the store
func (a *AnnotationStore) Add(ctx context.Context, anno chronograf.Annotation) (chronograf.Annotation, error) {
	return chronograf.Annotation{}, nil
}

// Delete removes the annotation from the store
func (a *AnnotationStore) Delete(ctx context.Context, anno chronograf.Annotation) error {
	return nil
}

// Update replaces annotatio
func (a *AnnotationStore) Update(context.Context, chronograf.Annotation) error {
	return nil
}

type value []interface{}

func (v value) Int64(idx int) (int64, error) {
	if idx >= len(v) {
		return 0, fmt.Errorf("index %d does not exist in values", idx)
	}
	i, ok := v[idx].(int64)
	if !ok {
		return 0, fmt.Errorf("value at index %d is not int64, but, %T", idx, v[idx])
	}
	return i, nil
}

func (v value) String(idx int) (string, error) {
	if idx >= len(v) {
		return "", fmt.Errorf("index %d does not exist in values", idx)
	}
	str, ok := v[idx].(string)
	if !ok {
		return "", fmt.Errorf("value at index %d is not string, but, %T", idx, v[idx])
	}
	return str, nil
}

type annotationResults []struct {
	Series []struct {
		Values []value `json:"values"`
	} `json:"series"`
}

// Annotations converts `SELECT "duration_ns", "text", "annoType", "id" FROM "chronograf"."autogen"."annotations"` to annotations
func (r *annotationResults) Annotations() (res []chronograf.Annotation, err error) {
	res = []chronograf.Annotation{}
	for _, u := range *r {
		for _, s := range u.Series {
			for _, v := range s.Values {
				anno := chronograf.Annotation{}
				if anno.Time, err = v.Int64(0); err != nil {
					return
				}

				if anno.Duration, err = v.Int64(1); err != nil {
					return
				}

				if anno.Text, err = v.String(2); err != nil {
					return
				}

				if anno.Type, err = v.String(3); err != nil {
					return
				}

				if anno.ID, err = v.String(4); err != nil {
					return
				}
				res = append(res, anno)
			}
		}
	}
	return res, err
}

// queryAnnotations queries the chronograf db and produces all annotations
// TODO: change this to nanoseconds and change to have start/end time
func (c *Client) queryAnnotations(ctx context.Context, query string) ([]chronograf.Annotation, error) {
	res, err := c.Query(ctx, chronograf.Query{
		Command: query,
		DB:      `chronograf`,
	})
	if err != nil {
		return nil, err
	}
	octets, err := res.MarshalJSON()
	if err != nil {
		return nil, err
	}

	results := annotationResults{}
	if err := json.Unmarshal(octets, &results); err != nil {
		return nil, err
	}
	return results.Annotations()
}

// allAnnotations returns all annotations from the chronograf.annotations measurement
func (c *Client) allAnnotations(ctx context.Context) ([]chronograf.Annotation, error) {
	return c.queryAnnotations(ctx, AllAnnotations)
}

// getAnnotations returns all annotations with id
func (c *Client) getAnnotations(ctx context.Context, id string) ([]chronograf.Annotation, error) {
	return c.queryAnnotations(ctx, fmt.Sprintf(GetAnnotationID, id))
}
