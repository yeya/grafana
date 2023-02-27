package api

import (
	"time"

	"github.com/prometheus/common/model"

	"github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
	"github.com/grafana/grafana/pkg/services/ngalert/models"
)

// AlertRuleFromProvisionedAlertRule converts definitions.ProvisionedAlertRule to models.AlertRule
func AlertRuleFromProvisionedAlertRule(a definitions.ProvisionedAlertRule) (models.AlertRule, error) {
	return models.AlertRule{
		ID:           a.ID,
		UID:          a.UID,
		OrgID:        a.OrgID,
		NamespaceUID: a.FolderUID,
		RuleGroup:    a.RuleGroup,
		Title:        a.Title,
		Condition:    a.Condition,
		Data:         AlertQueriesFromApiAlertQuery(a.Data),
		Updated:      a.Updated,
		NoDataState:  models.NoDataState(a.NoDataState),          // TODO there must be a validation
		ExecErrState: models.ExecutionErrorState(a.ExecErrState), // TODO there must be a validation
		For:          time.Duration(a.For),
		Annotations:  a.Annotations,
		Labels:       a.Labels,
		IsPaused:     a.IsPaused,
	}, nil
}

// ProvisionedAlertRuleFromAlertRule converts models.AlertRule to definitions.ProvisionedAlertRule and sets provided provenance status
func ProvisionedAlertRuleFromAlertRule(rule models.AlertRule, provenance models.Provenance) definitions.ProvisionedAlertRule {
	return definitions.ProvisionedAlertRule{
		ID:           rule.ID,
		UID:          rule.UID,
		OrgID:        rule.OrgID,
		FolderUID:    rule.NamespaceUID,
		RuleGroup:    rule.RuleGroup,
		Title:        rule.Title,
		For:          model.Duration(rule.For),
		Condition:    rule.Condition,
		Data:         AlertQueriesToApiAlertQueries(rule.Data),
		Updated:      rule.Updated,
		NoDataState:  definitions.NoDataState(rule.NoDataState),          // TODO there may be a validation
		ExecErrState: definitions.ExecutionErrorState(rule.ExecErrState), // TODO there may be a validation
		Annotations:  rule.Annotations,
		Labels:       rule.Labels,
		Provenance:   definitions.Provenance(provenance), // TODO validate enum conversion?
		IsPaused:     rule.IsPaused,
	}
}

// ProvisionedAlertRuleFromAlertRules converts a collection of models.AlertRule to definitions.ProvisionedAlertRules with provenance status models.ProvenanceNone
func ProvisionedAlertRuleFromAlertRules(rules []*models.AlertRule) definitions.ProvisionedAlertRules {
	result := make([]definitions.ProvisionedAlertRule, 0, len(rules))
	for _, r := range rules {
		result = append(result, ProvisionedAlertRuleFromAlertRule(*r, models.ProvenanceNone))
	}
	return result
}

// AlertQueriesFromApiAlertQuery converts a collection of definitions.AlertQuery to collection of models.AlertQuery
func AlertQueriesFromApiAlertQuery(queries []definitions.AlertQuery) []models.AlertQuery {
	result := make([]models.AlertQuery, 0, len(queries))
	for _, q := range queries {
		result = append(result, models.AlertQuery{
			RefID:     q.RefID,
			QueryType: q.QueryType,
			RelativeTimeRange: models.RelativeTimeRange{
				From: models.Duration(q.RelativeTimeRange.From),
				To:   models.Duration(q.RelativeTimeRange.To),
			},
			DatasourceUID: q.DatasourceUID,
			Model:         q.Model,
		})
	}
	return result
}

// AlertQueriesToApiAlertQueries converts a collection of models.AlertQuery to collection of definitions.AlertQuery
func AlertQueriesToApiAlertQueries(queries []models.AlertQuery) []definitions.AlertQuery {
	result := make([]definitions.AlertQuery, 0, len(queries))
	for _, q := range queries {
		result = append(result, definitions.AlertQuery{
			RefID:     q.RefID,
			QueryType: q.QueryType,
			RelativeTimeRange: definitions.RelativeTimeRange{
				From: definitions.Duration(q.RelativeTimeRange.From),
				To:   definitions.Duration(q.RelativeTimeRange.To),
			},
			DatasourceUID: q.DatasourceUID,
			Model:         q.Model,
		})
	}
	return result
}

func AlertRuleGroupFromApi(a definitions.AlertRuleGroup) (models.AlertRuleGroup, error) {
	ruleGroup := models.AlertRuleGroup{
		Title:     a.Title,
		FolderUID: a.FolderUID,
		Interval:  a.Interval,
	}
	for i := range a.Rules {
		converted, err := AlertRuleFromProvisionedAlertRule(a.Rules[i])
		if err != nil {
			return models.AlertRuleGroup{}, err
		}
		ruleGroup.Rules = append(ruleGroup.Rules, converted)
	}
	return ruleGroup, nil
}

func AlertRuleGroupToApi(d models.AlertRuleGroup) definitions.AlertRuleGroup {
	rules := make([]definitions.ProvisionedAlertRule, 0, len(d.Rules))
	for i := range d.Rules {
		rules = append(rules, ProvisionedAlertRuleFromAlertRule(d.Rules[i], d.Provenance))
	}
	return definitions.AlertRuleGroup{
		Title:     d.Title,
		FolderUID: d.FolderUID,
		Interval:  d.Interval,
		Rules:     rules,
	}
}
