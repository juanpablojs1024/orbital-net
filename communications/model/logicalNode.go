package model

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"satellite-coms/communications/internal/httpclient"
)

type LogicalNode struct {
	ID      string
	Name    string
	Message Instruction
	Memory  []string
	State   string
}

type Instruction struct {
	Origin    *LogicalNode
	Objective *LogicalNode
	Payload   string
}

func (ln *LogicalNode) SendInstruction(objLn *LogicalNode, msg string) {
	ln.Message = Instruction{Origin: ln, Objective: objLn, Payload: msg}
	ln.State = "wsc"
}

func GetLogicalNodes() []*LogicalNode {
	rawNodes := httpclient.FetchNodes()
	logicalNodes := []*LogicalNode{}

	for _, item := range rawNodes {
		node := item.(map[string]interface{})
		id := node["id"].(string)
		name := node["name"].(string)

		// Check if 'ports' exists and is a float64
		if portsVal, ok := node["ports"]; ok {
			ports := int(portsVal.(float64))

			// Create one LogicalNode per port
			for i := 1; i <= ports; i++ {
				logicalNodes = append(logicalNodes, &LogicalNode{
					ID:   fmt.Sprintf("%s:port%d", id, i),
					Name: fmt.Sprintf("%s (port %d)", name, i),
				})
			}
		} else {
			// Fallback if no 'ports' field – treat as one port
			logicalNodes = append(logicalNodes, &LogicalNode{
				ID:   fmt.Sprintf("%s:port1", id),
				Name: fmt.Sprintf("%s (port 1)", name),
			})
		}
	}
	return logicalNodes
}

func (ln *LogicalNode) GetPath(restrictions map[string]struct{}) []string {
	// Build restricted query string from map
	restrictedList := ""
	for id := range restrictions {
		if id != ln.ID && id != ln.Message.Objective.ID {
			if restrictedList != "" {
				restrictedList += ","
			}
			restrictedList += id
		}
	}

	url := fmt.Sprintf("http://localhost:8082/path?start=%s&end=%s&restricted=%s",
		ln.ID, ln.Message.Objective.ID, restrictedList)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to send request to Pathfinder: %v", err)
		return nil
	}
	defer resp.Body.Close()

	// Handle 404 (no path found)
	if resp.StatusCode == http.StatusNotFound {
		//log.Printf("🚫 No path found from %s to %s", ln.ID, ln.Message.Objective.ID)
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		//log.Printf("❌ Pathfinder returned unexpected status %d", resp.StatusCode)
		return nil
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to parse JSON response: %v", err)
		return nil
	}

	// Extract and convert "path" field to []string
	rawPath, ok := result["path"].([]interface{})
	if !ok {
		log.Println("Invalid path format in response")
		return nil
	}

	var path []string
	for _, v := range rawPath {
		if s, ok := v.(string); ok {
			path = append(path, s)
		}
	}

	return path
}

func GetLogicalNodeById(id string, logicalNodes []*LogicalNode) *LogicalNode {
	for _, ln := range logicalNodes {
		if ln.ID == id {
			return ln
		}
	}
	return &LogicalNode{}
}
