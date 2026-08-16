package apiserver

import (
	"encoding/json"
	"net/http"

	"github.com/Rish3666/ServeEz/internal/api"
)

// handleCostCompare prices a workload shape across providers. Requires the
// pricing engine to be attached via WithCost; otherwise returns 404.
func (s *Server) handleCostCompare(w http.ResponseWriter, r *http.Request) {
	if s.cost == nil {
		writeError(w, http.StatusNotFound, "cost comparer not enabled")
		return
	}
	var req api.CostCompareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid cost compare request")
		return
	}
	if req.VCPU <= 0 || req.MemGB <= 0 {
		writeError(w, http.StatusBadRequest, "vcpu and mem_gb required (> 0)")
		return
	}
	report, err := s.cost.Compare(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, report)
}
