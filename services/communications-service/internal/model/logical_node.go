package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	pbCommon "orbital-net/pkg/proto/common"
	pbPath "orbital-net/pkg/proto/pathfinder"
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

func (ln *LogicalNode) GetPath(pathfinderClient pbPath.PathfinderServiceClient, restrictions map[string]struct{}) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if ln.Message.Objective == nil {
		return nil
	}

	startPhysical := strings.Split(ln.ID, ":")[0]
	endPhysical := strings.Split(ln.Message.Objective.ID, ":")[0]

	req := &pbPath.PathRequest{
		StartNodeId: startPhysical,
		EndNodeId:   endPhysical,
	}

	resp, err := pathfinderClient.FindPath(ctx, req)
	if err != nil {
		return nil
	}

	if !resp.Found {
		return nil
	}

	return resp.PathNodeIds
}

func SyncLogicalNodes(current []*LogicalNode, simNodes []*pbCommon.Node) []*LogicalNode {
	existingMap := make(map[string]*LogicalNode)
	for _, ln := range current {
		existingMap[ln.ID] = ln
	}

	var updatedList []*LogicalNode

	for _, n := range simNodes {
		ports := int(n.PortQuantity)
		if ports <= 0 {
			ports = 1
		}

		for i := 1; i <= ports; i++ {
			id := fmt.Sprintf("%s:port%d", n.Id, i)

			if existing, ok := existingMap[id]; ok {
				updatedList = append(updatedList, existing)
			} else {
				updatedList = append(updatedList, &LogicalNode{
					ID:   id,
					Name: fmt.Sprintf("%s (port %d)", n.Name, i),
				})
			}
		}
	}
	return updatedList
}

func InitLogicalNodes(simNodes []*pbCommon.Node) []*LogicalNode {
	return SyncLogicalNodes(nil, simNodes)
}
