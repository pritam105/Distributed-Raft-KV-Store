package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"distributed-raft-kv-store/raft"
)

type PeerMap map[string]string

type HTTPTransport struct {
	peers  PeerMap
	client *http.Client
}

func NewHTTPTransport(peers PeerMap) *HTTPTransport {
	return &HTTPTransport{
		peers: peers,
		client: &http.Client{
			Timeout: 200 * time.Millisecond,
		},
	}
}

func (t *HTTPTransport) SendRequestVote(peerID string, args raft.RequestVoteArgs) (raft.RequestVoteReply, error) {
	var reply raft.RequestVoteReply
	if err := t.post(peerID, "/raft/request-vote", args, &reply); err != nil {
		return raft.RequestVoteReply{}, err
	}
	return reply, nil
}

func (t *HTTPTransport) SendAppendEntries(peerID string, args raft.AppendEntriesArgs) (raft.AppendEntriesReply, error) {
	var reply raft.AppendEntriesReply
	if err := t.post(peerID, "/raft/append-entries", args, &reply); err != nil {
		return raft.AppendEntriesReply{}, err
	}
	return reply, nil
}

func (t *HTTPTransport) post(peerID, path string, body, reply any) error {
	base, ok := t.peers[peerID]
	if !ok {
		return raft.ErrNotReachable{PeerID: peerID}
	}

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	resp, err := t.client.Post(base+path, "application/json", bytes.NewReader(data))
	if err != nil {
		return raft.ErrNotReachable{PeerID: peerID}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("peer %s returned %d", peerID, resp.StatusCode)
	}

	return json.NewDecoder(resp.Body).Decode(reply)
}
