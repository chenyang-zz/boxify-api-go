package tool

func cloneDescriptor(descriptor Descriptor) Descriptor {
	descriptor.Schema = cloneSchema(descriptor.Schema)
	descriptor.Annotations = cloneMap(descriptor.Annotations)
	return descriptor
}

func cloneSchema(schema Schema) Schema {
	schema.Parameters.Properties = cloneProperties(schema.Parameters.Properties)
	schema.Parameters.Required = cloneStrings(schema.Parameters.Required)
	schema.Parameters.AdditionalProperties = cloneAny(schema.Parameters.AdditionalProperties)
	if schema.Strict != nil {
		strict := *schema.Strict
		schema.Strict = &strict
	}
	return schema
}

func cloneProperties(values map[string]PropertySchema) map[string]PropertySchema {
	if values == nil {
		return nil
	}
	cloned := make(map[string]PropertySchema, len(values))
	for key, value := range values {
		cloned[key] = PropertySchema(cloneMap(value))
	}
	return cloned
}

func cloneSetDescriptor(descriptor SetDescriptor) SetDescriptor {
	descriptor.Tags = cloneStrings(descriptor.Tags)
	descriptor.Annotations = cloneMap(descriptor.Annotations)
	return descriptor
}

func cloneStrings(values []string) []string {
	if values == nil {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneMap(values map[string]any) map[string]any {
	if values == nil {
		return nil
	}
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneAny(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []string:
		return cloneStrings(typed)
	case []any:
		cloned := make([]any, len(typed))
		copy(cloned, typed)
		return cloned
	default:
		return value
	}
}
