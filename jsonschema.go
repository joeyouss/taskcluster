// Package jsonschema2go allows you to translate json schemas like this:
//
//  {
//    "definitions": {
//      "activities": {
//        "description": "A subset of all known human activities",
//        "type": "object",
//        "additionalProperties": false,
//        "properties": {
//          "snooker": {
//            "description": "The fine sport of snooker, invented in Madras around 1885",
//            "type": "boolean"
//          },
//          "cooking": {
//            "description": "The act of preparing food for consumption, typically involving the application of heat",
//            "type": "boolean"
//          }
//        },
//        "required": [
//          "cooking",
//          "snooker"
//        ]
//      }
//    },
//    "title": "person",
//    "description": "A member of the animal kingdom of planet Earth, dominant briefly around 13.8 billion years after the Big Bang",
//    "type": "object",
//    "additionalProperties": false,
//    "properties": {
//      "address": {
//        "description": "Where the person lives",
//        "type": "array",
//        "items": {
//          "type": "string"
//        }
//      },
//      "hobbies": {
//        "description": "Hobbies the person has",
//        "$ref": "#/definitions/activities"
//      },
//      "dislikes": {
//        "description": "Activities this person dislikes",
//        "$ref": "#/definitions/activities"
//      }
//    },
//    "required": [
//      "address"
//    ]
//  }
//
// into generated code like this:
//
//  // This source code file is AUTO-GENERATED by github.com/taskcluster/jsonschema2go
//
//  package main
//
//  type (
//  	// A subset of all known human activities
//  	Activities struct {
//
//  		// The act of preparing food for consumption, typically involving the application of heat
//  		Cooking bool `json:"cooking"`
//
//  		// The fine sport of snooker, invented in Madras around 1885
//  		Snooker bool `json:"snooker"`
//  	}
//
//  	// A member of the animal kingdom of planet Earth, dominant briefly around 13.8 billion years after the Big Bang
//  	Person struct {
//
//  		// Where the person lives
//  		Address []string `json:"address"`
//
//  		// Activities this person dislikes
//  		Dislikes Activities `json:"dislikes,omitempty"`
//
//  		// Hobbies the person has
//  		Hobbies Activities `json:"hobbies,omitempty"`
//  	}
//  )
//
// This then allows you to json.Unmarshal json data that conforms to a given
// schema into the generated types. By harnessing this library as part of your
// build process, you can ensure that your go types are always in sync with
// your json schemas.
package jsonschema2go

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/taskcluster/jsonschema2go/text"
)

type (
	// JsonSubSchema represents the data stored in a json subschema. Note that
	// all members are backed by pointers, so that nil value can signify
	// non-existence.  Otherwise we could not differentiate whether a zero
	// value is non-existence or actually the zero value. For example, if a
	// bool is false, we don't know if it was explictly set to false in the
	// json we read, or whether it was not given. Unmarshaling into a pointer
	// means pointer will be nil pointer if it wasn't read, or a pointer to
	// true/false if it was read from json.
	JsonSubSchema struct {
		AdditionalItems      *bool                  `json:"additionalItems,omitempty"`
		AdditionalProperties *AdditionalProperties  `json:"additionalProperties,omitempty"`
		AllOf                *Items                 `json:"allOf,omitempty"`
		AnyOf                *Items                 `json:"anyOf,omitempty"`
		Const                *interface{}           `json:"const,omitempty"`
		Default              *interface{}           `json:"default,omitempty"`
		Definitions          *Properties            `json:"definitions,omitempty"`
		Dependencies         map[string]*Dependency `json:"dependencies,omitempty"`
		Description          *string                `json:"description,omitempty"`
		Enum                 []interface{}          `json:"enum,omitempty"`
		ExclusiveMaximum     *bool                  `json:"exclusiveMaximum,omitempty"`
		ExclusiveMinimum     *bool                  `json:"exclusiveMinimum,omitempty"`
		Format               *string                `json:"format,omitempty"`
		ID                   *string                `json:"$id,omitempty"`
		Items                *JsonSubSchema         `json:"items,omitempty"`
		Maximum              *int                   `json:"maximum,omitempty"`
		MaxItems             *int                   `json:"maxItems,omitempty"`
		MaxLength            *int                   `json:"maxLength,omitempty"`
		MaxProperties        *int                   `json:"maxProperties,omitempty"`
		Minimum              *int                   `json:"minimum,omitempty"`
		MinItems             *int                   `json:"minItems,omitempty"`
		MinLength            *int                   `json:"minLength,omitempty"`
		MinProperties        *int                   `json:"minProperties,omitempty"`
		MultipleOf           *int                   `json:"multipleOf,omitempty"`
		OneOf                *Items                 `json:"oneOf,omitempty"`
		Pattern              *string                `json:"pattern,omitempty"`
		PatternProperties    *Properties            `json:"patternProperties,omitempty"`
		Properties           *Properties            `json:"properties,omitempty"`
		Ref                  *string                `json:"$ref,omitempty"`
		Required             []string               `json:"required,omitempty"`
		Schema               *string                `json:"$schema,omitempty"`
		Title                *string                `json:"title,omitempty"`
		Type                 *string                `json:"type,omitempty"`
		UniqueItems          *bool                  `json:"uniqueItems,omitempty"`

		// non-json fields used for sorting/tracking

		// TypeName is the name of the generated go type that represents this
		// JsonSubSchema, e.g. HawkSignatureAuthenticationRequest. If this
		// JsonSubSchema does not represent a struct (for example if it
		// represents a string, an int, an undefined object, etc), then
		// TypeName will be an empty string.
		TypeName string `json:"TYPE_NAME"`

		// If this schema is a schema inside a `properties` map of strings to
		// schemas of a parent json subschema, PropertyName will be the key
		// used in that parent schema to refer to this schema.
		//
		// If this schema is inside an array (under "items").
		//
		// Otherwise, PropertyName will be an empty string.
		PropertyName string         `json:"PROPERTY_NAME"`
		SourceURL    string         `json:"SOURCE_URL"`
		RefSchemaURL string         `json:"REF_SCHEMA_URL,omitempty"`
		RefSubSchema *JsonSubSchema `json:"REF_SUBSCHEMA,omitempty"`
		IsRequired   bool           `json:"IS_REQUIRED"`
	}

	Items struct {
		Items     []*JsonSubSchema
		SourceURL string
	}

	Properties struct {
		Properties          map[string]*JsonSubSchema
		MemberNames         map[string]string
		SortedPropertyNames []string
		SourceURL           string
	}

	AdditionalProperties struct {
		Boolean    *bool
		Properties *JsonSubSchema
	}

	Dependency struct {
		SchemaDependency   *JsonSubSchema
		PropertyDependency *[]string
	}

	canPopulate interface {
		postPopulate(*Job) error
		setSourceURL(string)
		prepare(*Job) error
	}

	NameGenerator func(name string, exported bool, blacklist map[string]bool) (identifier string)

	Job struct {
		Package              string
		ExportTypes          bool
		HideStructMembers    bool
		URLs                 []string
		result               *Result
		TypeNameGenerator    NameGenerator
		MemberNameGenerator  NameGenerator
		SkipCodeGen          bool
		TypeNameBlacklist    StringSet
		DisableNestedStructs bool
	}

	Result struct {
		SourceCode []byte
		SchemaSet  *SchemaSet
	}

	// SchemaSet contains the JsonSubSchemas objects read when performing a Job.
	SchemaSet struct {
		all       map[string]*JsonSubSchema
		used      map[string]*JsonSubSchema
		populated []canPopulate
		TypeNames StringSet
	}

	StringSet map[string]bool
)

// Ensure url contains "#" by adding it to end if needed
func sanitizeURL(url string) string {
	if strings.ContainsRune(url, '#') {
		return url
	}
	return url + "#"
}

func (schemaSet *SchemaSet) SubSchema(url string) *JsonSubSchema {
	return schemaSet.all[sanitizeURL(url)]
}

func (schemaSet *SchemaSet) SortedSanitizedURLs() []string {
	keys := make([]string, len(schemaSet.used))
	i := 0
	for k := range schemaSet.used {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

// May panic - this is recovered by fmt package, but care should be taken to
// capture panics when calling String() directly
func (subSchema JsonSubSchema) String() string {
	v, err := json.Marshal(subSchema)
	if err != nil {
		panic(err)
	}
	b, err := yaml.JSONToYAML(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (jsonSubSchema *JsonSubSchema) typeDefinition(disableNested bool, topLevel bool, extraPackages StringSet, rawMessageTypes StringSet) (comment, typ string) {
	// Ignore all other properties if this has a $ref, and only redirect to the referened schema.
	// See https://tools.ietf.org/html/draft-handrews-json-schema-01#section-8.3:
	//   `All other properties in a "$ref" object MUST be ignored.`
	if p := jsonSubSchema.RefSubSchema; p != nil {
		return p.typeDefinition(disableNested, topLevel, extraPackages, rawMessageTypes)
	}
	comment = "\n"
	if d := jsonSubSchema.Description; d != nil {
		comment += text.Indent(*d, "// ")
	}
	if comment[len(comment)-1:] != "\n" {
		comment += "\n"
	}
	if c := jsonSubSchema.Const; c != nil {
		comment += "//\n// Constant value: "
		switch (*c).(type) {
		case float64:
			comment += fmt.Sprintf("%v\n", *c)
		default:
			comment += fmt.Sprintf("%q\n", *c)
		}
	}
	if enum := jsonSubSchema.Enum; enum != nil {
		comment += "//\n// Possible values:\n"
		for _, i := range enum {
			switch i.(type) {
			case float64:
				comment += fmt.Sprintf("//   * %v\n", i)
			default:
				comment += fmt.Sprintf("//   * %q\n", i)
			}
		}
	}

	// Create comments for metadata in a single paragraph. Only start new
	// paragraph if we discover after inspecting all possible metadata, that
	// something has been specified. If there is no metadata, no need to create
	// a new paragraph.
	var metadata string
	if def := jsonSubSchema.Default; def != nil {
		var value string
		switch (*def).(type) {
		case bool:
			value = strconv.FormatBool((*def).(bool))
		case float64:
			value = strconv.FormatFloat((*def).(float64), 'g', -1, 64)
		default:
			v, err := json.MarshalIndent(*def, "", "  ")
			if err != nil {
				panic(fmt.Sprintf("couldn't marshal %+v", *def))
			}
			value = string(v)
		}
		indentedDefault := text.Indent(value+"\n", "//             ")
		metadata += "// Default:    " + indentedDefault[15:]
	}
	if regex := jsonSubSchema.Pattern; regex != nil {
		metadata += "// Syntax:     " + *regex + "\n"
	}
	if minItems := jsonSubSchema.MinLength; minItems != nil {
		metadata += "// Min length: " + strconv.Itoa(*minItems) + "\n"
	}
	if maxItems := jsonSubSchema.MaxLength; maxItems != nil {
		metadata += "// Max length: " + strconv.Itoa(*maxItems) + "\n"
	}
	if minimum := jsonSubSchema.Minimum; minimum != nil {
		metadata += "// Mininum:    " + strconv.Itoa(*minimum) + "\n"
	}
	if maximum := jsonSubSchema.Maximum; maximum != nil {
		metadata += "// Maximum:    " + strconv.Itoa(*maximum) + "\n"
	}
	if allOf := jsonSubSchema.AllOf; allOf != nil {
		metadata += "// All of:\n"
		for _, o := range allOf.Items {
			metadata += "//   * " + o.getTypeName() + "\n"
		}
	}
	if anyOf := jsonSubSchema.AnyOf; anyOf != nil {
		metadata += "// Any of:\n"
		for _, o := range anyOf.Items {
			metadata += "//   * " + o.getTypeName() + "\n"
		}
	}
	if oneOf := jsonSubSchema.OneOf; oneOf != nil {
		metadata += "// One of:\n"
		for _, o := range oneOf.Items {
			metadata += "//   * " + o.getTypeName() + "\n"
		}
	}
	// Here we check if metadata was specified, and only create new
	// paragraph (`//\n`) if something was.
	if len(metadata) > 0 {
		comment += "//\n" + metadata
	}
	typ = "json.RawMessage"
	if p := jsonSubSchema.Type; p != nil {
		typ = *p
	}
	switch typ {
	case "array":
		typ = "[]interface{}"
		if jsonSubSchema.Items != nil {
			arrayComment, arrayType := jsonSubSchema.Items.typeDefinition(disableNested, false, extraPackages, rawMessageTypes)
			typ = "[]" + arrayType
			// only add array comments if target schema is a primitive type
			if jsonSubSchema.Items.TargetSchema().TypeName == "" {
				// arrayComment already contains leading newline char (\n)
				comment += "//\n// Array items:" + arrayComment
			}
		}
	case "object":
		if jsonSubSchema.AnyOf != nil || jsonSubSchema.AllOf != nil || jsonSubSchema.OneOf != nil {
			typ = "json.RawMessage"
			break
		}
		ap := jsonSubSchema.AdditionalProperties
		noExtraProperties := ap != nil && ap.Boolean != nil && !*ap.Boolean
		if noExtraProperties {
			// If we are sure no additional properties are allowed, we can
			// generate a struct with all allowed property names.
			if !topLevel && disableNested {
				typ = jsonSubSchema.getTypeName()
			} else {
				typ = jsonSubSchema.Properties.AsStruct(disableNested, extraPackages, rawMessageTypes)
			}
		} else if ap != nil && ap.Properties != nil && jsonSubSchema.Properties == nil {
			// In the special case no properties have been specified, but
			// additionalProperties is an object, we can create a
			// map[string]<additionalProperties definition>.
			subComment, subType := ap.Properties.typeDefinition(disableNested, false, extraPackages, rawMessageTypes)
			typ = "map[string]" + subType
			// only add subcomments if target schema is a primitive type
			if ap.Properties.TargetSchema().TypeName == "" {
				// subComment already contains leading newline char (\n)
				comment += "//\n// Map entries:" + subComment
			}
		} else {
			// Either *arbitrarily structured* additional properties are
			// allowed, or the additional properties are of a fixed form, but
			// the explicitly listed properties may not conform to that form,
			// so fall back to the most general option to ensure it can hold
			// both listed properties and additional properties.
			if s := jsonSubSchema.Properties; s != nil {
				comment += "//\n// Defined properties:\n//\n"
				comment += text.Indent(s.AsStruct(disableNested, extraPackages, rawMessageTypes), "//  ") + "\n"
			}
			if ap != nil && ap.Properties != nil {
				comment += "//\n// Additional properties:\n"
				subComment, subType := ap.Properties.typeDefinition(disableNested, true, extraPackages, rawMessageTypes)
				comment += text.Indent(subComment, "//  ")
				comment += text.Indent(subType, "//  ") + "\n"
			} else {
				comment += "//\n// Additional properties allowed\n"
			}
			typ = "json.RawMessage"
		}
	case "number":
		typ = "float64"
	case "integer":
		typ = "int64"
	case "boolean":
		typ = "bool"
	// json type string maps to go type string, so only need to test case of when
	// string is a json date-time, so we can convert to go type Time...
	case "string":
		if f := jsonSubSchema.Format; f != nil {
			if *f == "date-time" {
				typ = "tcclient.Time"
				extraPackages["tcclient \"github.com/taskcluster/taskcluster-client-go\""] = true
			}
		}
	}

	if URL := jsonSubSchema.SourceURL; URL != "" {
		u, err := url.Parse(URL)
		if err == nil && u.Scheme != "file" {
			comment += "//\n// See " + URL + "\n"
		}
	}
	for strings.Index(comment, "\n//\n") == 0 {
		comment = "\n" + comment[4:]
	}

	switch typ {
	case "json.RawMessage":
		extraPackages["\"encoding/json\""] = true
		if topLevel {
			// Special case: we have here a top level RawMessage such as
			// queue.PostArtifactRequest - therefore need to implement
			// Marshal and Unmarshal methods. See:
			// http://play.golang.org/p/FKHSUmWVFD vs
			// http://play.golang.org/p/erjM6ptIYI
			extraPackages["\"errors\""] = true
			rawMessageTypes[jsonSubSchema.TypeName] = true
		}
	}
	return comment, typ
}

func (p Properties) String() string {
	result := ""
	for _, i := range p.SortedPropertyNames {
		result += "Property '" + i + "' =\n" + text.Indent(p.Properties[i].String(), "  ")
	}
	return result
}

func (p *Properties) prepare(job *Job) error {
	log.Printf("In PREPARE (properties): %v", p.SourceURL)
	for _, j := range p.SortedPropertyNames {
		if p.Properties[j].TargetSchema().Properties != nil {
			if job.DisableNestedStructs {
				job.add(p.Properties[j].TargetSchema())
			}
		}
	}
	return nil
}

func (p *Properties) postPopulate(job *Job) error {
	log.Printf("In POSTPOPULATE (properties): %v", p.SourceURL)
	job.result.SchemaSet.populated = append(job.result.SchemaSet.populated, p)
	// now all data should be loaded, let's sort the p.Properties
	if p.Properties != nil {
		p.SortedPropertyNames = make([]string, 0, len(p.Properties))
		for propertyName := range p.Properties {
			p.SortedPropertyNames = append(p.SortedPropertyNames, propertyName)
			// subschemas need to have SourceURL set
			p.Properties[propertyName].setSourceURL(p.SourceURL + "/" + propertyName)
			p.Properties[propertyName].PropertyName = propertyName
		}
		sort.Strings(p.SortedPropertyNames)
		members := make(StringSet, len(p.SortedPropertyNames))
		p.MemberNames = make(map[string]string, len(p.SortedPropertyNames))
		for _, j := range p.SortedPropertyNames {
			p.MemberNames[j] = job.MemberNameGenerator(j, !job.HideStructMembers, members)
			// subschemas also need to be triggered to postPopulate...
			err := p.Properties[j].postPopulate(job)
			if err != nil {
				return err
			}
		}
	} else {
		return fmt.Errorf("WEIRD - NO PROPERTIES in %v", p.SourceURL)
	}
	return nil
}

func (job *Job) SetTypeName(subSchema *JsonSubSchema, blacklist map[string]bool) {
	if r := subSchema.Ref; r != nil {
		log.Printf("Not setting type name for %v - has $ref to ", subSchema.SourceURL, subSchema.RefSubSchema.SourceURL)
		job.SetTypeName(subSchema.RefSubSchema, blacklist)
		return
	}
	if subSchema.TypeName != "" {
		log.Printf("Type name already set to '%v' for %v", subSchema.TypeName, subSchema.SourceURL)
		return
	}
	log.Printf("Setting type name for %v", subSchema.SourceURL)
	// Type names only need to be set for objects and arrays, everything else is a primitive type
	subSchema.TypeName = job.TypeNameGenerator(subSchema.TypeNameRaw(), job.ExportTypes, blacklist)
	if subSchema.Items != nil {
		log.Printf("Type %v is an array - will set type for items too...", subSchema.SourceURL)
		subSchema.Items.TargetSchema().PropertyName = subSchema.PropertyName + " entry"
		job.SetTypeName(subSchema.Items, blacklist)
	}
}

func (p *Properties) setSourceURL(url string) {
	p.SourceURL = url
}

func (i *Items) UnmarshalJSON(bytes []byte) (err error) {
	err = json.Unmarshal(bytes, &i.Items)
	return
}

func (p *Properties) UnmarshalJSON(bytes []byte) (err error) {
	err = json.Unmarshal(bytes, &p.Properties)
	return
}

func (d *Dependency) UnmarshalJSON(bytes []byte) (err error) {
	s, j := &[]string{}, new(JsonSubSchema)
	if err = json.Unmarshal(bytes, s); err == nil {
		d.PropertyDependency = s
		return
	}
	if err = json.Unmarshal(bytes, j); err == nil {
		d.SchemaDependency = j
	}
	return
}

func (aP *AdditionalProperties) UnmarshalJSON(bytes []byte) (err error) {
	b, p := new(bool), new(JsonSubSchema)
	if err = json.Unmarshal(bytes, b); err == nil {
		aP.Boolean = b
		return
	}
	if err = json.Unmarshal(bytes, p); err == nil {
		aP.Properties = p
	}
	return
}

func (aP AdditionalProperties) String() string {
	if aP.Boolean != nil {
		return strconv.FormatBool(*aP.Boolean)
	}
	return aP.Properties.String()
}

func (items Items) String() string {
	result := ""
	for i, j := range items.Items {
		result += fmt.Sprintf("Item '%v' =\n", i) + text.Indent(j.String(), "  ")
	}
	return result
}

func (items *Items) prepare(job *Job) error {
	log.Printf("In PREPARE (items): %v", items.SourceURL)
	for _, j := range (*items).Items {
		// add to schemas so we get a type generated for it in source code
		job.add(j.TargetSchema())
	}
	return nil
}

func (items *Items) postPopulate(job *Job) error {
	log.Printf("In POSTPOPULATE (items): %v", items.SourceURL)
	job.result.SchemaSet.populated = append(job.result.SchemaSet.populated, items)
	for i, j := range (*items).Items {
		j.setSourceURL(items.SourceURL + "[" + strconv.Itoa(i) + "]")
		err := j.postPopulate(job)
		if err != nil {
			return err
		}
	}
	return nil
}

func (subSchema *JsonSubSchema) TypeNameRaw() string {
	switch {
	case subSchema.RefSubSchema != nil:
		return subSchema.RefSubSchema.TypeNameRaw()
	case subSchema.Title != nil && *subSchema.Title != "" && len(*subSchema.Title) < 40:
		return *subSchema.Title
	case subSchema.PropertyName != "" && len(subSchema.PropertyName) < 40:
		return subSchema.PropertyName
	case subSchema.Description != nil && *subSchema.Description != "" && len(*subSchema.Description) < 40:
		return *subSchema.Description
	default:
		return "var"
	}
}

func (job *Job) add(subSchema *JsonSubSchema) {
	// if we have already included in the schema set, nothing to do...
	if _, ok := job.result.SchemaSet.used[subSchema.SourceURL]; ok {
		log.Printf("Not adding %v", subSchema.SourceURL)
		return
	}
	log.Printf("Adding %v", subSchema.SourceURL)
	job.result.SchemaSet.used[subSchema.SourceURL] = subSchema
	job.SetTypeName(subSchema, job.TypeNameBlacklist)
	job.result.SchemaSet.TypeNames[subSchema.TypeName] = true
}

func (items *Items) setSourceURL(url string) {
	items.SourceURL = url
}

func (subSchema *JsonSubSchema) postPopulateIfNotNil(canPopulate canPopulate, job *Job, suffix string) error {
	if reflect.ValueOf(canPopulate).IsValid() {
		if !reflect.ValueOf(canPopulate).IsNil() {
			canPopulate.setSourceURL(subSchema.SourceURL + suffix)
			err := canPopulate.postPopulate(job)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (subSchema *JsonSubSchema) link(job *Job) (err error) {
	if ref := subSchema.Ref; ref != nil && *ref != "" {
		subSchema.RefSubSchema = job.result.SchemaSet.all[subSchema.RefSchemaURL]
		if subSchema.RefSubSchema == nil {
			return fmt.Errorf("Subschema %v not loaded when updating %v", subSchema.RefSchemaURL, subSchema.SourceURL)
		}
		log.Printf("Linked %v to %v", subSchema.SourceURL, subSchema.RefSchemaURL)
	} else {
		log.Printf("Nothing to link in %v", subSchema.SourceURL)
	}
	return nil
}

func (subSchema *JsonSubSchema) prepare(job *Job) (err error) {
	log.Printf("In PREPARE (subschema): %v", subSchema.SourceURL)

	// If this subschema has Items (anyOf, oneOf, allOf) then we should "copy
	// down" properties from this schema into them, since they inherit the
	// values in this schema if they don't override them.
	subSchema.AllOf.MergeIn(subSchema, map[string]bool{"AllOf": true, "ID": true})
	subSchema.AnyOf.MergeIn(subSchema, map[string]bool{"AnyOf": true, "ID": true})
	subSchema.OneOf.MergeIn(subSchema, map[string]bool{"OneOf": true, "ID": true})

	subSchema.Type = subSchema.inferType()

	// Mark subschema properties that are in required list as being required (IsRequired property)
	for _, req := range subSchema.Required {
		if subSchema.Properties != nil {
			if subSubSchema, ok := subSchema.Properties.Properties[req]; ok {
				subSubSchema.IsRequired = true
			} else {
				panic(fmt.Sprintf("Schema %v has a required property %v but this property definition cannot be found", subSchema.SourceURL, req))
			}
		}
	}

	if job.DisableNestedStructs {
		// If this subschema is an array of objects, then add the object type to the top level types
		if subSchema.Items != nil && subSchema.Items.TargetSchema().Properties != nil {
			job.add(subSchema.Items.TargetSchema())
		}
		// If this subschema is a map of strings to objects, then add the object type to the top level types
		if subSchema.AdditionalProperties != nil && subSchema.AdditionalProperties.Properties != nil && subSchema.AdditionalProperties.Properties.TargetSchema().Properties != nil {
			job.add(subSchema.AdditionalProperties.Properties.TargetSchema())
		}
	}
	return nil
}

func (subSchema *JsonSubSchema) postPopulate(job *Job) (err error) {
	log.Printf("In POSTPOPULATE (subschema): %v", subSchema.SourceURL)
	job.result.SchemaSet.populated = append(job.result.SchemaSet.populated, subSchema)

	// Since setSourceURL(string) must be called before postPopulate(*Job), we
	// can rely on subSchema.SourceURL being already set.
	job.result.SchemaSet.all[subSchema.SourceURL] = subSchema

	// Call postPopulate on sub items of this schema...  Use an ARRAY not a MAP
	// so we can be sure subSchema.Definitions is processed before anything
	// that might reference it
	type Subcomponent struct {
		subPath string
		subItem canPopulate
	}

	subcomponents := []Subcomponent{
		{"/definitions", subSchema.Definitions},
		{"/allOf", subSchema.AllOf},
		{"/anyOf", subSchema.AnyOf},
		{"/oneOf", subSchema.OneOf},
		{"/items", subSchema.Items},
		{"/properties", subSchema.Properties},
	}
	if subSchema.AdditionalProperties != nil {
		subcomponents = append(subcomponents, Subcomponent{"/additionalProperties", subSchema.AdditionalProperties.Properties})
	}

	for _, s := range subcomponents {
		err = subSchema.postPopulateIfNotNil(s.subItem, job, s.subPath)
		if err != nil {
			return
		}
	}

	// If we have a $ref pointing to another schema, keep a reference so we can
	// discover TypeName later when we generate the type definition
	if ref := subSchema.Ref; ref != nil && *ref != "" {
		// relative references within current document are relatively simple...
		if strings.HasPrefix(*ref, "#") {
			subSchema.RefSchemaURL = subSchema.SourceURL[:strings.Index(subSchema.SourceURL, "#")] + *ref
			return
		}
		// looks like it's pointing to a different document - better make sure we've loaded/cached it
		// first need to determine if id property has been specified for a base url to resolve against...
		// see https://json-schema.org/understanding-json-schema/structuring.html#the-id-property
		var absURL string
		// note json schemas are nested, so we need to get to the root json schema of the document we're in
		// using the source URL we can strip off the internal path within the document (everything after '#')
		docURLRoot := strings.SplitN(subSchema.SourceURL, "#", 2)
		docBaseSchema := job.result.SchemaSet.SubSchema(docURLRoot[0] + "#")
		if id := docBaseSchema.ID; id != nil && *id != "" {
			// '$id' property is specified in doc, so let's use it!
			var refURL *url.URL
			refURL, err = url.Parse(*ref)
			if err != nil {
				return
			}
			var idURL *url.URL
			idURL, err = url.Parse(*id)
			if err != nil {
				return
			}
			absURL = idURL.ResolveReference(refURL).String()
		} else {
			// no '$id' property is specified, we must assume this is an absolute URL
			absURL = *subSchema.Ref
		}
		// make sure the doc is loaded (if in cache it won't be loaded again)
		subSchema.RefSubSchema, err = job.cacheJsonSchema(absURL)
		if err != nil {
			return
		}
		subSchema.RefSchemaURL = sanitizeURL(absURL)
	}
	return
}

func (subSchema *JsonSubSchema) TargetSchema() *JsonSubSchema {
	if ref := subSchema.RefSubSchema; ref != nil {
		return ref.TargetSchema()
	}
	return subSchema
}

// MergeIn copies attributes from subSchema into the subschemas in items.Items
// when they are not currently set.
func (items *Items) MergeIn(subSchema *JsonSubSchema, skipFields StringSet) {
	if items == nil || len(items.Items) == 0 {
		// nothing to do
		return
	}
	p := reflect.ValueOf(subSchema).Elem()
	// loop through all struct fields of Jsonsubschema
	for i := 0; i < p.NumField(); i++ {
		// don't copy fields that are blacklisted, or that aren't pointers
		if skipFields[p.Type().Field(i).Name] || p.Field(i).Kind() != reflect.Ptr {
			continue
		}
		// loop through all items (e.g. the list of oneOf schemas)
		for _, item := range items.Items {
			c := reflect.ValueOf(item).Elem()
			// only replace destination value if it is currently nil
			if destination, source := c.Field(i), p.Field(i); destination.IsNil() {

				// To copy the pointer, we would just:
				//   destination.Set(source)
				// However, we want to make copies of the entries, rather than
				// copy the pointers, so that future modifications of a copied
				// subschema won't update the source schema. Note: this is only
				// a top-level copy, not a deep copy, but is better than nothing.

				// dereference the pointer to get the value
				targetValue := reflect.Indirect(source)
				if targetValue.IsValid() {
					// create a new value to store it
					newValue := reflect.New(targetValue.Type()).Elem()
					// copy the value into the new value
					newValue.Set(targetValue)
					// create a new pointer to point to the new value
					newPointer := reflect.New(targetValue.Addr().Type()).Elem()
					// set that pointer to the address of the new value
					newPointer.Set(newValue.Addr())
					// copy the new pointer to the destination
					destination.Set(newPointer)
				}
			}
			// If we wanted to "move" instead of "copy", we could reset source
			// to nil with:
			//   source.Set(reflect.Zero(source.Type()))
		}
	}
}

func (subSchema *JsonSubSchema) setSourceURL(url string) {
	subSchema.SourceURL = url
}

func (job *Job) loadJsonSchema(URL string) (subSchema *JsonSubSchema, err error) {
	log.Printf("Loading %v", URL)
	var body io.ReadCloser
	if strings.HasPrefix(URL, "file://") {
		body, err = os.Open(URL[7 : len(URL)-1]) // need to strip trailing '#'
		if err != nil {
			return
		}
	} else {
		var u *url.URL
		u, err = url.Parse(URL)
		if err != nil {
			return
		}
		var resp *http.Response
		// TODO: may be better to use https://golang.org/pkg/net/http/#NewFileTransport here??
		switch u.Scheme {
		case "http", "https":
			resp, err = http.Get(URL)
			if err != nil {
				return subSchema, err
			}
			body = resp.Body
		default:
			return nil, fmt.Errorf("Unknown scheme '%s' for URL '%s'", u.Scheme, URL)
		}
	}
	defer body.Close()
	data, err := ioutil.ReadAll(body)
	if err != nil {
		return
	}
	// json is valid YAML, so we can safely convert, even if it is already json
	j, err := yaml.YAMLToJSON(data)
	if err != nil {
		return
	}
	subSchema = new(JsonSubSchema)
	err = json.Unmarshal(j, subSchema)
	if err != nil {
		return
	}
	subSchema.setSourceURL(sanitizeURL(URL))
	err = subSchema.postPopulate(job)
	return
}

func (job *Job) cacheJsonSchema(url string) (*JsonSubSchema, error) {
	// if url is not provided, there is nothing to download
	if url == "" {
		return nil, errors.New("Empty url in cacheJsonSchema")
	}
	sanitizedURL := sanitizeURL(url)
	// only fetch if we haven't fetched already...
	if _, ok := job.result.SchemaSet.all[sanitizedURL]; ok {
		log.Printf("Schema already cached: %v", url)
		return job.result.SchemaSet.SubSchema(sanitizedURL), nil
	}

	// The URL we load here could be a subschema nested inside a root document, e.g.
	// https://foo.com/schema.json#/definitions/bar/monkey
	// Therefore we need to load the schema from the URL portion up to the '#' char
	// (this char is guaranteed to be present) but return the particular subschema
	// that is located at the path underneath.

	// Containing document URL (sanitized URL up to and including '#' char)
	rootSchemaURL := sanitizedURL[:strings.Index(sanitizedURL, "#")+1]

	// Path to subschema from root of parent document (sanitized URL after '#' char)
	subschemaPath := sanitizedURL[strings.Index(sanitizedURL, "#")+1:]

	job.loadJsonSchema(rootSchemaURL)

	// check that the required subschema is contained in the document we loaded
	subschema, found := job.result.SchemaSet.all[sanitizedURL]
	if !found {
		return nil, fmt.Errorf("Subschema %v not found under URL %v", subschemaPath, rootSchemaURL)
	}
	return subschema, nil
}

// This is where we generate nested and compoound types in go to represent json payloads
// which are used as inputs and outputs for the REST API endpoints, and also for Pulse
// message bodies for the Exchange APIs.
// Returns the generated code content, and a map of keys of extra packages to import, e.g.
// a generated type might use time.Time, so if not imported, this would have to be added.
// using a map of strings -> bool to simulate a set - true => include
func generateGoTypes(disableNested bool, schemaSet *SchemaSet) (string, StringSet, StringSet) {
	extraPackages := make(StringSet)
	rawMessageTypes := make(StringSet)
	content := "type (" // intentionally no \n here since each type starts with one already
	// Loop through all json schemas that were found referenced inside the API json schemas...
	typeDefinitions := make(map[string]string)
	typeNames := make([]string, 0, len(schemaSet.used))
	for _, i := range schemaSet.used {
		log.Printf("Type name: '%v' - %v", i.getTypeName(), i.SourceURL)
		var newComment, newType string
		newComment, newType = i.typeDefinition(disableNested, true, extraPackages, rawMessageTypes)
		typeDefinitions[i.TypeName] = text.Indent(newComment+i.TypeName+" "+newType, "\t")
		typeNames = append(typeNames, i.getTypeName())
	}
	sort.Strings(typeNames)
	for _, t := range typeNames {
		content += typeDefinitions[t] + "\n"
	}
	return content + ")\n\n", extraPackages, rawMessageTypes
}

func (job *Job) Execute() (*Result, error) {
	// Generate normalised names for schemas. Keep a record of generated type
	// names, so that we don't reuse old names. Set acts like a set
	// of strings.
	job.result = new(Result)
	job.result.SchemaSet = &SchemaSet{
		all:       make(map[string]*JsonSubSchema),
		used:      make(map[string]*JsonSubSchema),
		populated: make([]canPopulate, 0, len(job.URLs)),
		TypeNames: make(StringSet),
	}
	if job.TypeNameBlacklist == nil {
		job.TypeNameBlacklist = make(StringSet)
	}
	if job.TypeNameGenerator == nil {
		job.TypeNameGenerator = text.GoIdentifierFrom
	}
	if job.MemberNameGenerator == nil {
		job.MemberNameGenerator = text.GoIdentifierFrom
	}
	for _, URL := range job.URLs {
		j, err := job.cacheJsonSchema(URL)
		if err != nil {
			return nil, err
		}
		// note we don't add inside cacheJsonSchema/loadJsonSchema
		// since we don't want to add e.g. top level items if only
		// definitions inside the schema are referenced
		job.add(j.TargetSchema())
	}
	for _, subSchema := range job.result.SchemaSet.all {
		err := subSchema.link(job)
		if err != nil {
			return nil, err
		}
	}
	for _, cp := range job.result.SchemaSet.populated {
		err := cp.prepare(job)
		if err != nil {
			return nil, err
		}
	}

	var err error
	if job.SkipCodeGen {
		return job.result, err
	}
	types, extraPackages, rawMessageTypes := generateGoTypes(job.DisableNestedStructs, job.result.SchemaSet)
	content := `// This source code file is AUTO-GENERATED by github.com/taskcluster/jsonschema2go

package ` + job.Package + `

`
	extraPackagesContent := ""
	for j, k := range extraPackages {
		if k {
			extraPackagesContent += text.Indent(""+j+"\n", "\t")
		}
	}

	if extraPackagesContent != "" {
		content += `import (
` + extraPackagesContent + `)

`
	}
	content += types
	content += jsonRawMessageImplementors(rawMessageTypes)
	// format it
	job.result.SourceCode, err = format.Source([]byte(content))
	if err != nil {
		err = fmt.Errorf("Formatting error: %v\n%v", err, content)
	}
	return job.result, err
	// imports should be good, so no need to run
	// https://godoc.org/golang.org/x/tools/imports#Process
}

func jsonRawMessageImplementors(rawMessageTypes StringSet) string {
	// first sort the order of the rawMessageTypes since when we rebuild, we
	// don't want to generate functions in a different order and introduce
	// diffs against the previous version
	sortedRawMessageTypes := make([]string, len(rawMessageTypes))
	i := 0
	for goType := range rawMessageTypes {
		sortedRawMessageTypes[i] = goType
		i++
	}
	sort.Strings(sortedRawMessageTypes)
	content := ""
	for _, goType := range sortedRawMessageTypes {
		content += `

	// MarshalJSON calls json.RawMessage method of the same name. Required since
	// ` + goType + ` is of type json.RawMessage...
	func (this *` + goType + `) MarshalJSON() ([]byte, error) {
		x := json.RawMessage(*this)
		return (&x).MarshalJSON()
	}

	// UnmarshalJSON is a copy of the json.RawMessage implementation.
	func (this *` + goType + `) UnmarshalJSON(data []byte) error {
		if this == nil {
			return errors.New("` + goType + `: UnmarshalJSON on nil pointer")
		}
		*this = append((*this)[0:0], data...)
		return nil
	}`
	}
	return content
}

func (s *Properties) AsStruct(disableNested bool, extraPackages StringSet, rawMessageTypes StringSet) (typ string) {
	typ = fmt.Sprintf("struct {\n")
	if s != nil {
		for _, j := range s.SortedPropertyNames {
			// recursive call to build structs inside structs
			var subComment, subType string
			subMember := s.MemberNames[j]
			subComment, subType = s.Properties[j].typeDefinition(disableNested, false, extraPackages, rawMessageTypes)
			jsonStructTagOptions := ""
			if !s.Properties[j].IsRequired {
				jsonStructTagOptions = ",omitempty"
			}
			// struct member name and type, as part of struct definition
			typ += text.Indent(fmt.Sprintf("%v%v %v `json:\"%v%v\"`", subComment, subMember, subType, j, jsonStructTagOptions), "\t") + "\n"
		}
	}
	typ += "}"
	return
}

func (jsonSubSchema *JsonSubSchema) getTypeName() string {
	if jsonSubSchema.Ref != nil {
		return jsonSubSchema.RefSubSchema.getTypeName()
	}
	return jsonSubSchema.TypeName
}

// inferType is a cheeky little function that tries to set the type, if it can
// infer it from other information, such as if all OneOf subschemas share the
// same type, for example.
func (subSchema *JsonSubSchema) inferType() *string {

	// 1) If already set, nothing to do...
	if subSchema.Type != nil {
		return subSchema.Type
	}

	// 2) See if we can infer from existence of `properties` or `items`
	var inferredType string
	switch {
	case subSchema.Properties != nil:
		inferredType = "object"
	case subSchema.Items != nil:
		inferredType = "array"
	}
	if inferredType != "" {
		return &inferredType
	}

	// 3) If all items in subSchema.AllOf/subSchema.AnyOf/subSchema.OneOf have
	// same type, we can infer that is the type
	for _, items := range []*Items{
		subSchema.AllOf,
		subSchema.AnyOf,
		subSchema.OneOf,
	} {
		if items != nil {
			for _, subSubSchema := range items.Items {
				subType := subSubSchema.inferType()
				if subType == nil {
					return nil
				}
				if inferredType == "" {
					inferredType = *subType
					continue
				}
				if inferredType != *subType {
					return nil
				}
			}
			return &inferredType
		}
	}

	// 4) If const is set, infer from that
	if subSchema.Const != nil {
		return jsonSchemaTypeFromValue(*subSchema.Const)
	}

	// 5) If an enum, see if all entries have same type
	for _, enumItem := range subSchema.Enum {
		enumType := jsonSchemaTypeFromValue(enumItem)
		if inferredType == "" {
			inferredType = *enumType
			continue
		}
		if inferredType != *enumType {
			return nil
		}
	}
	if inferredType != "" {
		return &inferredType
	}

	// 6) Cannot infer type
	return nil
}

func jsonSchemaTypeFromValue(v interface{}) *string {
	var inferredType string
	switch t := v.(type) {
	case bool:
		inferredType = "boolean"
	case float64:
		inferredType = "number"
	case string:
		inferredType = "string"
	case []interface{}:
		inferredType = "array"
	case map[string]interface{}:
		inferredType = "object"
	case nil:
		inferredType = "null"
	default:
		log.Fatalf("What the? %v", t)
	}
	return &inferredType
}
