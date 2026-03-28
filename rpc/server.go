package rpc

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"distributed-raft-kv-store/raft"
)

type Server struct {
	node   *raft.Node
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
