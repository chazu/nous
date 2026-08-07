package domains

units: [
	{
		name:  "ValidConfiguration"
		worth: 500
		isA: ["UnaryPred", "Pred", "Op", "Anything"]
		domain: ["Anything"]
		range: ["Anything"]
		arity: 1
		defn:  "config-valid?"
	},
	{
		name:  "ValidConfigurationSchema"
		worth: 500
		isA: ["UnaryPred", "Pred", "Op", "Anything"]
		domain: ["Anything"]
		range: ["Anything"]
		arity: 1
		defn:  "config-schema-valid?"
	},
	{
		name:  "ConfigRepairKappa"
		worth: 600
		isA: ["PrimitiveConfigurationRepair", "UnaryOp", "Op", "Anything"]
		domain: ["Configuration"]
		range: ["Configuration"]
		arity:       1
		repairKey:   "service_port"
		repairValue: "443"
		defn:        "\"service_port\" \"443\" config-set"
	},
	{
		name:  "ConfigRepairLambda"
		worth: 600
		isA: ["PrimitiveConfigurationRepair", "UnaryOp", "Op", "Anything"]
		domain: ["Configuration"]
		range: ["Configuration"]
		arity:       1
		repairKey:   "replicas"
		repairValue: "2"
		defn:        "\"replicas\" \"2\" config-set"
	},
	{
		name:  "ConfigRepairMu"
		worth: 600
		isA: ["PrimitiveConfigurationRepair", "UnaryOp", "Op", "Anything"]
		domain: ["Configuration"]
		range: ["Configuration"]
		arity:       1
		repairKey:   "admin_public"
		repairValue: "false"
		defn:        "\"admin_public\" \"false\" config-set"
	},
	{
		name:  "ConfigRepairNu"
		worth: 600
		isA: ["PrimitiveConfigurationRepair", "UnaryOp", "Op", "Anything"]
		domain: ["Configuration"]
		range: ["Configuration"]
		arity:       1
		repairKey:   "redirect_http"
		repairValue: "true"
		defn:        "\"redirect_http\" \"true\" config-set"
	},
	{
		name:  "ConfigRepairXi"
		worth: 600
		isA: ["PrimitiveConfigurationRepair", "UnaryOp", "Op", "Anything"]
		domain: ["Configuration"]
		range: ["Configuration"]
		arity:       1
		repairKey:   "tls"
		repairValue: "false"
		defn:        "\"tls\" \"false\" config-set"
	},
	{
		name:  "ConfigRepairOmicron"
		worth: 600
		isA: ["PrimitiveConfigurationRepair", "UnaryOp", "Op", "Anything"]
		domain: ["Configuration"]
		range: ["Configuration"]
		arity:       1
		repairKey:   "environment"
		repairValue: "development"
		defn:        "\"environment\" \"development\" config-set"
	},
]
