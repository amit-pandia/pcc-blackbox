package main

import (
	"fmt"
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"testing"
	"time"
)

// get topics
func testGetTopic(t *testing.T) {
	if topics, err := pcc.Pcc.GetTopics(); err == nil {
		if len(topics) == 0 {
			t.Fatal("unable to fetch topic names")
		} else {
			fmt.Println(fmt.Sprintf("get topic %+v", topics))
		}
	} else {
		t.Fatal(err)
	}
}

// get topics schema
func testGetTopicSchema(t *testing.T) {
	if topics, err := pcc.Pcc.GetTopics(); err == nil {
		for _, topic := range topics {
			if schema, err := pcc.Pcc.GetSchema(topic); err == nil {
				fmt.Println(fmt.Sprintf("there are %d schemas for topic %s", len(schema), topic))
			} else {
				t.Fatal(err)
			}
		}
	} else {
		t.Fatal(err)
	}
}

// get a sample and check the content
func testMonitorSample(t *testing.T) {
	if nodes, err := pcc.Pcc.GetNodes(); err == nil {
		node := (*nodes)[0]
		if data, err := pcc.Pcc.GetLiveSample("cpu", node.Id); err == nil {
			fmt.Println(fmt.Sprintf("read cpu sample for node %d %+v", node.Id, data))
			name := (*data)["node"]
			if name != node.Name {
				t.Fatal("get the wrong sample")
			}
		} else {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}
}

// get history
func testMonitorHistory(t *testing.T) {
	to := time.Now().Unix()
	from := to - (1000 * 60 * 100)

	if data, err := pcc.Pcc.GetHistory("cpu", from, to); err == nil {
		fmt.Println(fmt.Sprintf("read cpu history %+v", data)) // nothing to do
	} else {
		t.Fatal(err)
	}
}
