package domains

units: [
	{
		name:  "ConfigurationRepairVocabulary"
		worth: 700
		isA: ["Vocabulary", "Anything"]
		dslExtension: "configrepair"
		english:      "Bounded typed-configuration repair vocabulary"
	},
	{name: "Configuration", worth: 650, isA: ["Anything"]},
	{name: "ConfigurationSchema", worth: 700, isA: ["Anything"]},
	{name: "ConfigurationRepairExample", worth: 650, isA: ["Anything"]},
	{name: "PrimitiveConfigurationRepair", worth: 650, isA: ["UnaryOp", "Op", "Anything"]},
	{name: "CompositeConfigurationRepair", worth: 650, isA: ["UnaryOp", "Op", "Anything"]},
	{name: "ConfigurationRepairResult", worth: 500, isA: ["Configuration", "Anything"]},
	{name: "ConfigurationRepairObservation", worth: 450, isA: ["Anything"]},
	{name: "ConfigurationRepairEvidence", worth: 500, isA: ["Anything"]},
	{name: "ConfigurationRepairSchema", worth: 750, isA: ["Anything"]},
	{
		name:  "ServiceSchemaV1"
		worth: 700
		isA: ["ConfigurationSchema", "Anything"]
		data: [
			"field:environment:string",
			"field:tls:bool",
			"field:service_port:int:1:65535",
			"field:replicas:int:0:10",
			"field:admin_public:bool",
			"field:redirect_http:bool",
			"required:environment",
			"required:tls",
			"required:service_port",
			"required:replicas",
			"required:admin_public",
			"required:redirect_http",
			"protected:environment",
			"protected:tls",
			"eq-if:tls=true,service_port=443",
			"min-if:environment=production,replicas=2",
			"eq-if:environment=production,admin_public=false",
		]
	},
	{
		name:  "GatewaySchemaV1"
		worth: 700
		isA: ["ConfigurationSchema", "Anything"]
		data: [
			"field:environment:string",
			"field:tls:bool",
			"field:service_port:int:1:65535",
			"field:replicas:int:0:10",
			"field:admin_public:bool",
			"field:redirect_http:bool",
			"field:route_count:int:0:100",
			"required:environment",
			"required:tls",
			"required:service_port",
			"required:replicas",
			"required:admin_public",
			"required:redirect_http",
			"required:route_count",
			"protected:environment",
			"protected:tls",
			"eq-if:tls=true,service_port=443",
			"min-if:environment=production,replicas=2",
			"eq-if:environment=production,admin_public=false",
		]
	},
	{
		name:  "ServiceExampleA"
		worth: 650
		isA: ["ConfigurationRepairExample", "Anything"]
		schema: "ServiceSchemaV1"
		configuration: [
			"environment=production", "tls=true", "service_port=80",
			"replicas=0", "admin_public=true", "redirect_http=false",
		]
	},
	{
		name:  "ServiceExampleB"
		worth: 650
		isA: ["ConfigurationRepairExample", "Anything"]
		schema: "ServiceSchemaV1"
		configuration: [
			"environment=production", "tls=true", "service_port=443",
			"replicas=0", "admin_public=true", "redirect_http=false",
		]
	},
	{
		name:  "GatewayExampleC"
		worth: 650
		isA: ["ConfigurationRepairExample", "Anything"]
		schema: "GatewaySchemaV1"
		configuration: [
			"environment=production", "tls=true", "service_port=80",
			"replicas=2", "admin_public=true", "redirect_http=false", "route_count=12",
		]
	},
	{
		name:  "GatewayExampleD"
		worth: 650
		isA: ["ConfigurationRepairExample", "Anything"]
		schema: "GatewaySchemaV1"
		configuration: [
			"environment=production", "tls=true", "service_port=80",
			"replicas=2", "admin_public=false", "redirect_http=false", "route_count=5",
		]
	},
]
