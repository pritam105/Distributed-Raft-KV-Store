package rpc

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"distributed-raft-kv-store/kv"
	"distributed-raft-kv-store/raft"
	"distributed-raft-kv-store/storage"
)

type Server struct {
	node   *raft.Node
	store  *kv.Store
	engine *gin.Engine
}

func NewServer(node *raft.Node) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	s := &Server{node: node, engine: r}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.engine
}

// RegisterKVRoutes adds GET/PUT/DELETE /v1/keys/:key with leader check.
func (s *Server) RegisterKVRoutes(store *kv.Store) {
	s.store = store
	s.engine.GET("/v1/keys/:key", s.handleGet)
	s.engine.PUT("/v1/keys/:key", s.handlePut)
	s.engine.DELETE("/v1/keys/:key", s.handleDelete)
}

func (s *Server) routes() {
	s.engine.POST("/raft/request-vote", s.handleRequestVote)
	s.engine.POST("/raft/append-entries", s.handleAppendEntries)
	s.engine.GET("/healthz", s.handleHealth)
	s.engine.GET("/status", s.handleStatus)
	s.engine.GET("/metrics", s.handleMetrics)
}

func (s *Server) handleRequestVote(c *gin.Context) {
	var args raft.RequestVoteArgs
	if err := c.ShouldBindJSON(&args); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reply := s.node.HandleRequestVote(args)
	c.JSON(http.StatusOK, reply)
}

func (s *Server) handleAppendEntries(c *gin.Context) {
	var args raft.AppendEntriesArgs
	if err := c.ShouldBindJSON(&args); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reply := s.node.HandleAppendEntries(args)
	c.JSON(http.StatusOK, reply)
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"nodeID":   s.node.ID(),
		"state":    s.node.State().String(),
		"isLeader": s.node.IsLeader(),
		"leaderID": s.node.LeaderID(),
		"term":     s.node.Term(),
	})
}

func (s *Server) handleMetrics(c *gin.Context) {
	m := s.node.Metrics.Snapshot()
	c.JSON(http.StatusOK, gin.H{
		"nodeID":           s.node.ID(),
		"state":            s.node.State().String(),
		"term":             s.node.Term(),
		"electionsStarted": m.ElectionsStarted,
		"votesGranted":     m.VotesGranted,
		"heartbeatsSent":   m.HeartbeatsSent,
		"heartbeatsRecvd":  m.HeartbeatsRecvd,
		"currentLeaderID":  m.CurrentLeaderID,
	})
}

// --- KV handlers ---

func (s *Server) handleGet(c *gin.Context) {
	key := c.Param("key")
	value, found, err := s.store.Get(key)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"key": key, "value": value, "found": true})
}

type kvPutRequest struct {
	Value string `json:"value"`
}

func (s *Server) handlePut(c *gin.Context) {
	if !s.node.IsLeader() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":    "not the leader",
			"leaderID": s.node.LeaderID(),
		})
		return
	}

	key := c.Param("key")
	var req kvPutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	cmd, err := json.Marshal(storage.Entry{Op: storage.OpUpsert, Key: key, Value: req.Value})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := s.node.Replicate(cmd); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": key, "value": req.Value})
}

func (s *Server) handleDelete(c *gin.Context) {
	if !s.node.IsLeader() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":    "not the leader",
			"leaderID": s.node.LeaderID(),
		})
		return
	}

	key := c.Param("key")
	cmd, err := json.Marshal(storage.Entry{Op: storage.OpDelete, Key: key})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := s.node.Replicate(cmd); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
