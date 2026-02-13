// Copyright (c) HashiCorp, Inc.

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ctrlplanedev/terraform-provider-ctrlplane/internal/api"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &PolicyResource{}
var _ resource.ResourceWithImportState = &PolicyResource{}
var _ resource.ResourceWithConfigure = &PolicyResource{}
var _ resource.ResourceWithValidateConfig = &PolicyResource{}

func NewPolicyResource() resource.Resource {
	return &PolicyResource{}
}

type PolicyResource struct {
	workspace *api.WorkspaceClient
}

func (r *PolicyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_policy"
}

func (r *PolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func (r *PolicyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	workspace, ok := req.ProviderData.(*api.WorkspaceClient)
	if !ok {
		resp.Diagnostics.AddError("Invalid provider data", "The provider data is not a *api.WorkspaceClient")
		return
	}

	r.workspace = workspace
}

func (r *PolicyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The ID of the policy",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The name of the policy",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "The description of the policy",
			},
			"metadata": schema.MapAttribute{
				Optional:    true,
				Description: "The metadata of the policy",
				ElementType: types.StringType,
			},
			"priority": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The priority of the policy (higher is evaluated first)",
				Default:     int64default.StaticInt64(0),
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the policy is enabled",
				Default:     booldefault.StaticBool(true),
			},
			"selector": schema.StringAttribute{
				Required:    true,
				Description: "CEL expression for matching release targets. Use \"true\" to match all targets.",
			},
		},
		Blocks: map[string]schema.Block{
			"version_cooldown": schema.ListNestedBlock{
				Description: "Version cooldown rules",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"created_at": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Rule creation timestamp",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"id": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Rule ID",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"duration": schema.StringAttribute{
							Required:    true,
							Description: "Minimum duration between deployments (e.g., \"1h\")",
						},
					},
				},
			},
			"deployment_window": schema.ListNestedBlock{
				Description: "Deployment window rules",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"created_at": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Rule creation timestamp",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"id": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Rule ID",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"duration_minutes": schema.Int64Attribute{
							Required:    true,
							Description: "Duration of each window in minutes",
						},
						"rrule": schema.StringAttribute{
							Required:    true,
							Description: "RFC 5545 recurrence rule for window starts",
						},
						"timezone": schema.StringAttribute{
							Optional:    true,
							Description: "IANA timezone for the recurrence rule",
						},
						"allow_window": schema.BoolAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Allow deployments during the window (deny when false)",
							Default:     booldefault.StaticBool(true),
						},
					},
				},
			},
			"deployment_dependency": schema.ListNestedBlock{
				Description: "Deployment dependency rules",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"created_at": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Rule creation timestamp",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"id": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Rule ID",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"depends_on_selector": schema.StringAttribute{
							Required:    true,
							Description: "CEL expression to match upstream deployment(s) that must have a successful release before this deployment can proceed",
						},
					},
				},
			},
			"verification": schema.ListNestedBlock{
				Description: "Verification rules",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"created_at": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Rule creation timestamp",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"id": schema.StringAttribute{
							Optional:    true,
							Computed:    true,
							Description: "Rule ID",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"trigger_on": schema.StringAttribute{
							Optional:    true,
							Description: "When to trigger verification (e.g., \"jobSuccess\")",
						},
					},
					Blocks: map[string]schema.Block{
						"metric": schema.ListNestedBlock{
							Description: "Verification metrics",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"name": schema.StringAttribute{
										Required:    true,
										Description: "Metric name",
									},
									"interval": schema.StringAttribute{
										Required:    true,
										Description: "Interval between measurements (e.g., \"30s\")",
									},
									"count": schema.Int64Attribute{
										Required:    true,
										Description: "Number of measurements to take",
									},
								},
								Blocks: map[string]schema.Block{
									"success": schema.SingleNestedBlock{
										Description: "Success condition",
										Attributes: map[string]schema.Attribute{
											"condition": schema.StringAttribute{
												Required:    true,
												Description: "CEL expression to evaluate success",
											},
											"threshold": schema.Int64Attribute{
												Optional:    true,
												Description: "Minimum consecutive successes required",
											},
										},
									},
									"failure": schema.SingleNestedBlock{
										Description: "Failure condition",
										Attributes: map[string]schema.Attribute{
											"condition": schema.StringAttribute{
												Optional:    true,
												Description: "CEL expression to evaluate failure",
											},
											"threshold": schema.Int64Attribute{
												Optional:    true,
												Description: "Consecutive failures before failing",
											},
										},
									},
									"datadog": schema.SingleNestedBlock{
										Description: "Datadog metric provider configuration",
										Attributes: map[string]schema.Attribute{
											"site": schema.StringAttribute{
												Optional:    true,
												Description: "Datadog site URL (e.g., us5.datadoghq.com)",
											},
											"interval": schema.StringAttribute{
												Optional:    true,
												Description: "Provider interval (e.g., \"1m\")",
											},
											"queries": schema.MapAttribute{
												Required:    true,
												Description: "Datadog metric queries",
												ElementType: types.StringType,
											},
											"api_key": schema.StringAttribute{
												Required:    true,
												Description: "Datadog API key",
												Sensitive:   true,
											},
											"app_key": schema.StringAttribute{
												Required:    true,
												Description: "Datadog application key",
												Sensitive:   true,
											},
											"aggregator": schema.StringAttribute{
												Optional:    true,
												Description: "Datadog aggregator (e.g., \"avg\")",
											},
											"formula": schema.StringAttribute{
												Optional:    true,
												Description: "Datadog formula",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *PolicyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data PolicyResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.Selector.IsNull() || data.Selector.IsUnknown() || data.Selector.ValueString() == "" {
		resp.Diagnostics.AddError("Invalid policy configuration", "The selector attribute must be set to a CEL expression.")
		return
	}
}

func (r *PolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PolicyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, diags := policyRulesFromModel(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	priority := int(defaultInt64(data.Priority, 0))
	enabled := defaultBool(data.Enabled, true)
	selector := data.Selector.ValueString()

	policyID := uuid.NewString()
	data.ID = types.StringValue(policyID)
	ensurePolicyIDs(&data, nil)
	ensurePolicyRuleCreatedAt(&data, nil)

	requestBody := policyRequestPayload{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
		Metadata:    stringMapPointer(data.Metadata),
		Priority:    &priority,
		Enabled:     &enabled,
		Rules:       &rules,
		Selector:    &selector,
	}

	setPolicyIDOnRules(&requestBody, policyID)

	body, err := json.Marshal(requestBody)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create policy", err.Error())
		return
	}

	policyResp, err := r.workspace.Client.RequestPolicyCreationWithBodyWithResponse(
		ctx,
		r.workspace.ID.String(),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create policy", err.Error())
		return
	}

	if policyResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to create policy", formatResponseError(policyResp.StatusCode(), policyResp.Body))
		return
	}

	if policyResp.JSON202 == nil || policyResp.JSON202.Id == "" {
		resp.Diagnostics.AddError("Failed to create policy", "Empty response from server")
		return
	}

	createdID := policyResp.JSON202.Id
	data.ID = types.StringValue(createdID)

	if createdID != policyID {
		updateBody := policyRequestPayload{
			Name:        data.Name.ValueString(),
			Description: data.Description.ValueStringPointer(),
			Metadata:    stringMapPointer(data.Metadata),
			Priority:    &priority,
			Enabled:     &enabled,
			Rules:       &rules,
			Selector:    &selector,
		}
		setPolicyIDOnRules(&updateBody, createdID)
		updatePayload, err := json.Marshal(updateBody)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update policy", err.Error())
			return
		}
		updateResp, err := r.workspace.Client.RequestPolicyUpsertWithBodyWithResponse(
			ctx,
			r.workspace.ID.String(),
			createdID,
			"application/json",
			bytes.NewReader(updatePayload),
		)
		if err != nil {
			resp.Diagnostics.AddError("Failed to update policy", err.Error())
			return
		}
		if updateResp.StatusCode() != http.StatusAccepted {
			resp.Diagnostics.AddError("Failed to update policy", formatResponseError(updateResp.StatusCode(), updateResp.Body))
			return
		}
	}

	err = waitForResource(ctx, func() (bool, error) {
		getResp, err := r.workspace.Client.GetPolicyWithResponse(ctx, r.workspace.ID.String(), data.ID.ValueString())
		if err != nil {
			return false, err
		}
		switch getResp.StatusCode() {
		case http.StatusOK:
			return true, nil
		case http.StatusNotFound:
			return false, nil
		default:
			return false, fmt.Errorf("unexpected status %d", getResp.StatusCode())
		}
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create policy", fmt.Sprintf("Resource not available after creation: %s", err.Error()))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *PolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyResp, err := r.workspace.Client.GetPolicyWithResponse(ctx, r.workspace.ID.String(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read policy", err.Error())
		return
	}

	switch policyResp.StatusCode() {
	case http.StatusOK:
		if policyResp.JSON200 == nil {
			resp.Diagnostics.AddError("Failed to read policy", "Empty response from server")
			return
		}
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	default:
		resp.Diagnostics.AddError("Failed to read policy", formatResponseError(policyResp.StatusCode(), policyResp.Body))
		return
	}

	policy := policyResp.JSON200
	data.ID = types.StringValue(policy.Id)
	data.Name = types.StringValue(policy.Name)
	data.Description = descriptionValue(policy.Description)
	data.Metadata = stringMapValue(&policy.Metadata)
	data.Priority = types.Int64Value(int64(policy.Priority))
	data.Enabled = types.BoolValue(policy.Enabled)

	data.Selector = types.StringValue(policy.Selector)

	rules, diags := policyRulesToModel(policy.Rules)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.VersionCooldown = rules.VersionCooldown
	data.DeploymentWindow = rules.DeploymentWindow
	data.DeploymentDependency = rules.DeploymentDependency
	data.Verification = rules.Verification

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data PolicyResourceModel
	var state PolicyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = state.ID
	ensurePolicyIDs(&data, &state)
	ensurePolicyRuleCreatedAt(&data, &state)

	rules, diags := policyRulesFromModel(data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	priority := int(defaultInt64(data.Priority, 0))
	enabled := defaultBool(data.Enabled, true)
	selector := data.Selector.ValueString()

	requestBody := policyRequestPayload{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueStringPointer(),
		Metadata:    stringMapPointer(data.Metadata),
		Priority:    &priority,
		Enabled:     &enabled,
		Rules:       &rules,
		Selector:    &selector,
	}

	setPolicyIDOnRules(&requestBody, data.ID.ValueString())

	body, err := json.Marshal(requestBody)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update policy", err.Error())
		return
	}

	policyResp, err := r.workspace.Client.RequestPolicyUpsertWithBodyWithResponse(
		ctx,
		r.workspace.ID.String(),
		data.ID.ValueString(),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update policy", err.Error())
		return
	}

	if policyResp.StatusCode() != http.StatusAccepted {
		resp.Diagnostics.AddError("Failed to update policy", formatResponseError(policyResp.StatusCode(), policyResp.Body))
		return
	}

	if policyResp.JSON202 == nil {
		resp.Diagnostics.AddError("Failed to update policy", "Empty response from server")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *PolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PolicyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyResp, err := r.workspace.Client.RequestPolicyDeletionWithResponse(ctx, r.workspace.ID.String(), data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete policy", err.Error())
		return
	}

	switch policyResp.StatusCode() {
	case http.StatusAccepted, http.StatusNoContent:
		return
	case http.StatusNotFound:
		resp.State.RemoveResource(ctx)
		return
	default:
		resp.Diagnostics.AddError("Failed to delete policy", formatResponseError(policyResp.StatusCode(), policyResp.Body))
		return
	}
}

type PolicyResourceModel struct {
	ID                   types.String                 `tfsdk:"id"`
	Name                 types.String                 `tfsdk:"name"`
	Description          types.String                 `tfsdk:"description"`
	Metadata             types.Map                    `tfsdk:"metadata"`
	Priority             types.Int64                  `tfsdk:"priority"`
	Enabled              types.Bool                   `tfsdk:"enabled"`
	Selector             types.String                 `tfsdk:"selector"`
	VersionCooldown      []PolicyVersionCooldown      `tfsdk:"version_cooldown"`
	DeploymentWindow     []PolicyDeploymentWindow     `tfsdk:"deployment_window"`
	DeploymentDependency []PolicyDeploymentDependency `tfsdk:"deployment_dependency"`
	Verification         []PolicyVerificationRule     `tfsdk:"verification"`
}

type PolicyVersionCooldown struct {
	CreatedAt types.String `tfsdk:"created_at"`
	ID        types.String `tfsdk:"id"`
	Duration  types.String `tfsdk:"duration"`
}

type PolicyDeploymentWindow struct {
	CreatedAt       types.String `tfsdk:"created_at"`
	ID              types.String `tfsdk:"id"`
	DurationMinutes types.Int64  `tfsdk:"duration_minutes"`
	Rrule           types.String `tfsdk:"rrule"`
	Timezone        types.String `tfsdk:"timezone"`
	AllowWindow     types.Bool   `tfsdk:"allow_window"`
}

type PolicyDeploymentDependency struct {
	CreatedAt         types.String `tfsdk:"created_at"`
	ID                types.String `tfsdk:"id"`
	DependsOnSelector types.String `tfsdk:"depends_on_selector"`
}

type PolicyVerificationRule struct {
	CreatedAt types.String               `tfsdk:"created_at"`
	ID        types.String               `tfsdk:"id"`
	TriggerOn types.String               `tfsdk:"trigger_on"`
	Metric    []PolicyVerificationMetric `tfsdk:"metric"`
}

type PolicyVerificationMetric struct {
	Name     types.String                 `tfsdk:"name"`
	Interval types.String                 `tfsdk:"interval"`
	Count    types.Int64                  `tfsdk:"count"`
	Success  *PolicyVerificationCondition `tfsdk:"success"`
	Failure  *PolicyVerificationCondition `tfsdk:"failure"`
	Datadog  *PolicyDatadogProvider       `tfsdk:"datadog"`
}

type PolicyVerificationCondition struct {
	Condition types.String `tfsdk:"condition"`
	Threshold types.Int64  `tfsdk:"threshold"`
}

type PolicyDatadogProvider struct {
	Site       types.String `tfsdk:"site"`
	Interval   types.String `tfsdk:"interval"`
	Queries    types.Map    `tfsdk:"queries"`
	ApiKey     types.String `tfsdk:"api_key"`
	AppKey     types.String `tfsdk:"app_key"`
	Aggregator types.String `tfsdk:"aggregator"`
	Formula    types.String `tfsdk:"formula"`
}

type policyRulesModel struct {
	VersionCooldown      []PolicyVersionCooldown
	DeploymentWindow     []PolicyDeploymentWindow
	DeploymentDependency []PolicyDeploymentDependency
	Verification         []PolicyVerificationRule
}

type policyRequestPayload struct {
	Description *string              `json:"description,omitempty"`
	Enabled     *bool                `json:"enabled,omitempty"`
	Metadata    *map[string]string   `json:"metadata,omitempty"`
	Name        string               `json:"name"`
	Priority    *int                 `json:"priority,omitempty"`
	Rules       *[]policyRequestRule `json:"rules,omitempty"`
	Selector    *string              `json:"selector,omitempty"`
}

type policyRequestRule struct {
	CreatedAt            string                        `json:"createdAt"`
	Id                   string                        `json:"id"`
	DeploymentDependency *api.DeploymentDependencyRule `json:"deploymentDependency,omitempty"`
	DeploymentWindow     *api.DeploymentWindowRule     `json:"deploymentWindow,omitempty"`
	Verification         *api.VerificationRule         `json:"verification,omitempty"`
	VersionCooldown      *api.VersionCooldownRule      `json:"versionCooldown,omitempty"`
	PolicyId             *string                       `json:"policyId,omitempty"`
}

func selectorValueSet(value types.String) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueString() != ""
}

func selectorIDValue(value types.String) string {
	if selectorValueSet(value) {
		return value.ValueString()
	}
	return uuid.NewString()
}

func createdAtValue(value types.String) string {
	if selectorValueSet(value) {
		return value.ValueString()
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func formatDuration(value time.Duration) string {
	if value%time.Hour == 0 {
		return fmt.Sprintf("%dh", int64(value/time.Hour))
	}
	if value%time.Minute == 0 {
		return fmt.Sprintf("%dm", int64(value/time.Minute))
	}
	if value%time.Second == 0 {
		return fmt.Sprintf("%ds", int64(value/time.Second))
	}
	return value.String()
}

func int64ValueSet(value types.Int64) bool {
	return !value.IsNull() && !value.IsUnknown()
}

func defaultInt64(value types.Int64, fallback int64) int64 {
	if value.IsNull() || value.IsUnknown() {
		return fallback
	}
	return value.ValueInt64()
}

func defaultBool(value types.Bool, fallback bool) bool {
	if value.IsNull() || value.IsUnknown() {
		return fallback
	}
	return value.ValueBool()
}

func policyRulesFromModel(data PolicyResourceModel) ([]policyRequestRule, diag.Diagnostics) {
	var diags diag.Diagnostics
	var rules []policyRequestRule

	for _, cooldown := range data.VersionCooldown {
		id := selectorIDValue(cooldown.ID)
		seconds, err := parseDurationSeconds(cooldown.Duration)
		if err != nil {
			diags.AddError("Invalid version cooldown duration", err.Error())
			continue
		}
		rules = append(rules, policyRequestRule{
			CreatedAt: createdAtValue(cooldown.CreatedAt),
			Id:        id,
			VersionCooldown: &api.VersionCooldownRule{
				IntervalSeconds: int32(seconds),
			},
		})
	}

	for _, window := range data.DeploymentWindow {
		id := selectorIDValue(window.ID)
		allowWindow := defaultBool(window.AllowWindow, true)
		rule := api.DeploymentWindowRule{
			AllowWindow:     allowWindow,
			DurationMinutes: int32(window.DurationMinutes.ValueInt64()),
			Rrule:           window.Rrule.ValueString(),
		}
		if selectorValueSet(window.Timezone) {
			timezone := window.Timezone.ValueString()
			rule.Timezone = &timezone
		}
		rules = append(rules, policyRequestRule{
			CreatedAt:        createdAtValue(window.CreatedAt),
			Id:               id,
			DeploymentWindow: &rule,
		})
	}

	for _, dep := range data.DeploymentDependency {
		id := selectorIDValue(dep.ID)
		rules = append(rules, policyRequestRule{
			CreatedAt: createdAtValue(dep.CreatedAt),
			Id:        id,
			DeploymentDependency: &api.DeploymentDependencyRule{
				DependsOn: dep.DependsOnSelector.ValueString(),
			},
		})
	}

	for _, verification := range data.Verification {
		id := selectorIDValue(verification.ID)
		verificationRule, err := policyVerificationRuleFromModel(verification)
		if err != nil {
			diags.AddError("Invalid verification rule", err.Error())
			continue
		}
		rules = append(rules, policyRequestRule{
			CreatedAt:    createdAtValue(verification.CreatedAt),
			Id:           id,
			Verification: verificationRule,
		})
	}

	return rules, diags
}

func policyVerificationRuleFromModel(model PolicyVerificationRule) (*api.VerificationRule, error) {
	if len(model.Metric) == 0 {
		return nil, fmt.Errorf("verification rule must define at least one metric")
	}

	metrics := make([]api.VerificationMetricSpec, 0, len(model.Metric))
	for _, metric := range model.Metric {
		spec, err := policyVerificationMetricFromModel(metric)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, spec)
	}

	rule := &api.VerificationRule{
		Metrics: metrics,
	}

	if selectorValueSet(model.TriggerOn) {
		triggerOn := api.VerificationRuleTriggerOn(model.TriggerOn.ValueString())
		rule.TriggerOn = &triggerOn
	}

	return rule, nil
}

func policyVerificationMetricFromModel(model PolicyVerificationMetric) (api.VerificationMetricSpec, error) {
	if model.Success == nil {
		return api.VerificationMetricSpec{}, fmt.Errorf("metric success block is required")
	}
	if model.Datadog == nil {
		return api.VerificationMetricSpec{}, fmt.Errorf("metric datadog block is required")
	}

	intervalSeconds, err := parseDurationSeconds(model.Interval)
	if err != nil {
		return api.VerificationMetricSpec{}, err
	}

	count := int(model.Count.ValueInt64())
	if count <= 0 {
		return api.VerificationMetricSpec{}, fmt.Errorf("metric count must be greater than zero")
	}

	successCondition := model.Success.Condition.ValueString()
	if successCondition == "" {
		return api.VerificationMetricSpec{}, fmt.Errorf("success condition must be set")
	}

	provider, err := policyDatadogProviderFromModel(*model.Datadog)
	if err != nil {
		return api.VerificationMetricSpec{}, err
	}

	spec := api.VerificationMetricSpec{
		Name:             model.Name.ValueString(),
		IntervalSeconds:  int32(intervalSeconds),
		Count:            count,
		SuccessCondition: successCondition,
		Provider:         provider,
	}

	if int64ValueSet(model.Success.Threshold) {
		threshold := int(model.Success.Threshold.ValueInt64())
		spec.SuccessThreshold = &threshold
	}
	if model.Failure != nil && selectorValueSet(model.Failure.Condition) {
		condition := model.Failure.Condition.ValueString()
		spec.FailureCondition = &condition
	}
	if model.Failure != nil && int64ValueSet(model.Failure.Threshold) {
		threshold := int(model.Failure.Threshold.ValueInt64())
		spec.FailureThreshold = &threshold
	}

	return spec, nil
}

func policyDatadogProviderFromModel(model PolicyDatadogProvider) (api.MetricProvider, error) {
	queries, err := mapStringValue(model.Queries)
	if err != nil {
		return api.MetricProvider{}, fmt.Errorf("invalid provider queries: %w", err)
	}

	datadog := api.DatadogMetricProvider{
		Type:    api.Datadog,
		ApiKey:  model.ApiKey.ValueString(),
		AppKey:  model.AppKey.ValueString(),
		Queries: queries,
	}

	if selectorValueSet(model.Site) {
		site := model.Site.ValueString()
		datadog.Site = &site
	}
	if selectorValueSet(model.Interval) {
		intervalSeconds, err := parseDurationSeconds(model.Interval)
		if err != nil {
			return api.MetricProvider{}, err
		}
		seconds := intervalSeconds
		datadog.IntervalSeconds = &seconds
	}
	if selectorValueSet(model.Aggregator) {
		aggregator := api.DatadogMetricProviderAggregator(model.Aggregator.ValueString())
		datadog.Aggregator = &aggregator
	}
	if selectorValueSet(model.Formula) {
		formula := model.Formula.ValueString()
		datadog.Formula = &formula
	}

	var provider api.MetricProvider
	if err := provider.FromDatadogMetricProvider(datadog); err != nil {
		return api.MetricProvider{}, err
	}

	return provider, nil
}

func policyRulesToModel(rules []api.PolicyRule) (policyRulesModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := policyRulesModel{}

	for _, rule := range rules {
		if rule.VersionCooldown != nil {
			duration := time.Duration(rule.VersionCooldown.IntervalSeconds) * time.Second
			result.VersionCooldown = append(result.VersionCooldown, PolicyVersionCooldown{
				CreatedAt: types.StringValue(rule.CreatedAt),
				ID:        types.StringValue(rule.Id),
				Duration:  types.StringValue(formatDuration(duration)),
			})
		}
		if rule.DeploymentWindow != nil {
			model := PolicyDeploymentWindow{
				CreatedAt:       types.StringValue(rule.CreatedAt),
				ID:              types.StringValue(rule.Id),
				DurationMinutes: types.Int64Value(int64(rule.DeploymentWindow.DurationMinutes)),
				Rrule:           types.StringValue(rule.DeploymentWindow.Rrule),
				Timezone:        types.StringNull(),
				AllowWindow:     types.BoolValue(rule.DeploymentWindow.AllowWindow),
			}
			if rule.DeploymentWindow.Timezone != nil {
				model.Timezone = types.StringValue(*rule.DeploymentWindow.Timezone)
			}
			result.DeploymentWindow = append(result.DeploymentWindow, model)
		}
		if rule.DeploymentDependency != nil {
			result.DeploymentDependency = append(result.DeploymentDependency, PolicyDeploymentDependency{
				CreatedAt:         types.StringValue(rule.CreatedAt),
				ID:                types.StringValue(rule.Id),
				DependsOnSelector: types.StringValue(rule.DeploymentDependency.DependsOn),
			})
		}
		if rule.Verification != nil {
			verification, err := policyVerificationRuleToModel(rule.Verification)
			if err != nil {
				diags.AddError("Invalid verification rule", err.Error())
				continue
			}
			verification.CreatedAt = types.StringValue(rule.CreatedAt)
			verification.ID = types.StringValue(rule.Id)
			result.Verification = append(result.Verification, verification)
		}
	}

	return result, diags
}

func ensurePolicyIDs(plan *PolicyResourceModel, state *PolicyResourceModel) {
	mergeCooldownIDs(plan.VersionCooldown, cooldownListFromState(state))
	mergeWindowIDs(plan.DeploymentWindow, windowListFromState(state))
	mergeDeploymentDependencyIDs(plan.DeploymentDependency, deploymentDependencyListFromState(state))
	mergeVerificationIDs(plan.Verification, verificationListFromState(state))
}

func ensurePolicyRuleCreatedAt(plan *PolicyResourceModel, state *PolicyResourceModel) {
	mergeCooldownCreatedAt(plan.VersionCooldown, cooldownListFromState(state))
	mergeWindowCreatedAt(plan.DeploymentWindow, windowListFromState(state))
	mergeDeploymentDependencyCreatedAt(plan.DeploymentDependency, deploymentDependencyListFromState(state))
	mergeVerificationCreatedAt(plan.Verification, verificationListFromState(state))
}

func setPolicyIDOnRules(request *policyRequestPayload, policyID string) {
	if request == nil || request.Rules == nil {
		return
	}

	for i := range *request.Rules {
		if (*request.Rules)[i].PolicyId == nil || *(*request.Rules)[i].PolicyId == "" {
			value := policyID
			(*request.Rules)[i].PolicyId = &value
		}
	}
}

func cooldownListFromState(state *PolicyResourceModel) []PolicyVersionCooldown {
	if state == nil {
		return nil
	}
	return state.VersionCooldown
}

func windowListFromState(state *PolicyResourceModel) []PolicyDeploymentWindow {
	if state == nil {
		return nil
	}
	return state.DeploymentWindow
}

func verificationListFromState(state *PolicyResourceModel) []PolicyVerificationRule {
	if state == nil {
		return nil
	}
	return state.Verification
}

func deploymentDependencyListFromState(state *PolicyResourceModel) []PolicyDeploymentDependency {
	if state == nil {
		return nil
	}
	return state.DeploymentDependency
}

func mergeCooldownIDs(plan []PolicyVersionCooldown, state []PolicyVersionCooldown) {
	for i := range plan {
		if selectorValueSet(plan[i].ID) {
			continue
		}
		if i < len(state) && selectorValueSet(state[i].ID) {
			plan[i].ID = state[i].ID
			continue
		}
		plan[i].ID = types.StringValue(uuid.NewString())
	}
}

func mergeCooldownCreatedAt(plan []PolicyVersionCooldown, state []PolicyVersionCooldown) {
	for i := range plan {
		if selectorValueSet(plan[i].CreatedAt) {
			continue
		}
		if i < len(state) && selectorValueSet(state[i].CreatedAt) {
			plan[i].CreatedAt = state[i].CreatedAt
			continue
		}
		plan[i].CreatedAt = types.StringValue(time.Now().UTC().Format(time.RFC3339))
	}
}

func mergeWindowIDs(plan []PolicyDeploymentWindow, state []PolicyDeploymentWindow) {
	for i := range plan {
		if selectorValueSet(plan[i].ID) {
			continue
		}
		if i < len(state) && selectorValueSet(state[i].ID) {
			plan[i].ID = state[i].ID
			continue
		}
		plan[i].ID = types.StringValue(uuid.NewString())
	}
}

func mergeWindowCreatedAt(plan []PolicyDeploymentWindow, state []PolicyDeploymentWindow) {
	for i := range plan {
		if selectorValueSet(plan[i].CreatedAt) {
			continue
		}
		if i < len(state) && selectorValueSet(state[i].CreatedAt) {
			plan[i].CreatedAt = state[i].CreatedAt
			continue
		}
		plan[i].CreatedAt = types.StringValue(time.Now().UTC().Format(time.RFC3339))
	}
}

func mergeDeploymentDependencyIDs(plan []PolicyDeploymentDependency, state []PolicyDeploymentDependency) {
	for i := range plan {
		if selectorValueSet(plan[i].ID) {
			continue
		}
		if i < len(state) && selectorValueSet(state[i].ID) {
			plan[i].ID = state[i].ID
			continue
		}
		plan[i].ID = types.StringValue(uuid.NewString())
	}
}

func mergeDeploymentDependencyCreatedAt(plan []PolicyDeploymentDependency, state []PolicyDeploymentDependency) {
	for i := range plan {
		if selectorValueSet(plan[i].CreatedAt) {
			continue
		}
		if i < len(state) && selectorValueSet(state[i].CreatedAt) {
			plan[i].CreatedAt = state[i].CreatedAt
			continue
		}
		plan[i].CreatedAt = types.StringValue(time.Now().UTC().Format(time.RFC3339))
	}
}

func mergeVerificationIDs(plan []PolicyVerificationRule, state []PolicyVerificationRule) {
	for i := range plan {
		if selectorValueSet(plan[i].ID) {
			continue
		}
		if i < len(state) && selectorValueSet(state[i].ID) {
			plan[i].ID = state[i].ID
			continue
		}
		plan[i].ID = types.StringValue(uuid.NewString())
	}
}

func mergeVerificationCreatedAt(plan []PolicyVerificationRule, state []PolicyVerificationRule) {
	for i := range plan {
		if selectorValueSet(plan[i].CreatedAt) {
			continue
		}
		if i < len(state) && selectorValueSet(state[i].CreatedAt) {
			plan[i].CreatedAt = state[i].CreatedAt
			continue
		}
		plan[i].CreatedAt = types.StringValue(time.Now().UTC().Format(time.RFC3339))
	}
}

func policyVerificationRuleToModel(rule *api.VerificationRule) (PolicyVerificationRule, error) {
	model := PolicyVerificationRule{
		TriggerOn: types.StringNull(),
		Metric:    make([]PolicyVerificationMetric, 0, len(rule.Metrics)),
	}

	if rule.TriggerOn != nil {
		model.TriggerOn = types.StringValue(string(*rule.TriggerOn))
	}

	for _, metric := range rule.Metrics {
		m, err := policyVerificationMetricToModel(metric)
		if err != nil {
			return PolicyVerificationRule{}, err
		}
		model.Metric = append(model.Metric, m)
	}

	return model, nil
}

func policyVerificationMetricToModel(metric api.VerificationMetricSpec) (PolicyVerificationMetric, error) {
	provider, err := metric.Provider.AsDatadogMetricProvider()
	if err != nil {
		return PolicyVerificationMetric{}, err
	}

	model := PolicyVerificationMetric{
		Name:     types.StringValue(metric.Name),
		Interval: types.StringValue((time.Duration(metric.IntervalSeconds) * time.Second).String()),
		Count:    types.Int64Value(int64(metric.Count)),
		Success: &PolicyVerificationCondition{
			Condition: types.StringValue(metric.SuccessCondition),
			Threshold: types.Int64Null(),
		},
		Failure: nil,
		Datadog: &PolicyDatadogProvider{},
	}

	if metric.SuccessThreshold != nil {
		model.Success.Threshold = types.Int64Value(int64(*metric.SuccessThreshold))
	}
	if metric.FailureCondition != nil || metric.FailureThreshold != nil {
		model.Failure = &PolicyVerificationCondition{
			Condition: types.StringNull(),
			Threshold: types.Int64Null(),
		}
		if metric.FailureCondition != nil {
			model.Failure.Condition = types.StringValue(*metric.FailureCondition)
		}
		if metric.FailureThreshold != nil {
			model.Failure.Threshold = types.Int64Value(int64(*metric.FailureThreshold))
		}
	}

	model.Datadog.Site = types.StringNull()
	if provider.Site != nil {
		model.Datadog.Site = types.StringValue(*provider.Site)
	}
	model.Datadog.Interval = types.StringNull()
	if provider.IntervalSeconds != nil {
		model.Datadog.Interval = types.StringValue((time.Duration(*provider.IntervalSeconds) * time.Second).String())
	}
	model.Datadog.Queries = types.MapNull(types.StringType)
	if len(provider.Queries) > 0 {
		result, _ := types.MapValueFrom(context.Background(), types.StringType, provider.Queries)
		model.Datadog.Queries = result
	}
	model.Datadog.ApiKey = types.StringValue(provider.ApiKey)
	model.Datadog.AppKey = types.StringValue(provider.AppKey)
	model.Datadog.Aggregator = types.StringNull()
	if provider.Aggregator != nil {
		model.Datadog.Aggregator = types.StringValue(string(*provider.Aggregator))
	}
	model.Datadog.Formula = types.StringNull()
	if provider.Formula != nil {
		model.Datadog.Formula = types.StringValue(*provider.Formula)
	}

	return model, nil
}

func parseDurationSeconds(value types.String) (int64, error) {
	if value.IsNull() || value.IsUnknown() {
		return 0, fmt.Errorf("duration must be set")
	}
	raw := value.ValueString()
	duration, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q", raw)
	}
	if duration < 0 {
		return 0, fmt.Errorf("duration must be non-negative")
	}
	if duration%time.Second != 0 {
		return 0, fmt.Errorf("duration %q must be a whole number of seconds", raw)
	}
	return int64(duration.Seconds()), nil
}

func parseDurationMinutes(value types.String) (int64, error) {
	if value.IsNull() || value.IsUnknown() {
		return 0, fmt.Errorf("duration must be set")
	}
	raw := value.ValueString()
	duration, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid duration %q", raw)
	}
	if duration < 0 {
		return 0, fmt.Errorf("duration must be non-negative")
	}
	if duration%time.Minute != 0 {
		return 0, fmt.Errorf("duration %q must be a whole number of minutes", raw)
	}
	return int64(duration.Minutes()), nil
}

func mapStringValue(value types.Map) (map[string]string, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, fmt.Errorf("map must be set")
	}
	var decoded map[string]string
	diags := value.ElementsAs(context.Background(), &decoded, false)
	if diags.HasError() {
		return nil, fmt.Errorf("invalid map value")
	}
	return decoded, nil
}
