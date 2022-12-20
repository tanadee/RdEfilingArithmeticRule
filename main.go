package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"os"
)

type Field struct {
	Name         string   `json:"name,omitempty"`
	Indicator    string   `json:"indicator,omitempty"`
	Multiplier   *float64 `json:"multiplier,omitempty"`
	DefaultValue *float64 `json:"defaultValue,omitempty"`
}

type Rule struct {
	Expression struct {
		Field Field   `json:"field,omitempty"`
		Sum   []Field `json:"sum,omitempty"`
	} `json:"expression,omitempty"`
	ErrorCode        string   `json:"errorCode,omitempty"`
	DecimalPrecision int      `json:"decimalPrecision"`
	EnableFlags      []string `json:"enableFlags,omitempty"`
	RuleFlags        []string `json:"ruleFlags,omitempty"`
}

var appendFieldName = []string{".total", ".exemption", ".liable"}
var zeroFloat64 float64

var threeColumnRuleFilepath = flag.String("threeColumnRuleFilepath", `jsonRule_three_column.txt`, "a path to file rule that need to be expand to 3 columns rules")
var normalRuleFilepath = flag.String("normalRuleFilepath", `jsonRule_normal.txt`, "a path to file rule without 3 columns rule")
var outputRuleFilepath = flag.String("outputRuleFilepath", `jsonRule.out.txt`, "an output file path")

func normalRule(rules []Rule) ([]Rule, error) {
	var result []Rule
	for _, r := range rules {
		r.Expression.Field.DefaultValue = &zeroFloat64
		for i := range r.Expression.Sum {
			r.Expression.Sum[i].DefaultValue = &zeroFloat64
		}
		setDefault(&r)
		result = append(result, r)
	}
	return result, nil
}

func threeColumnRule(rules []Rule) ([]Rule, error) {
	allFieldName := map[string]bool{}
	var result []Rule
	for _, rule := range rules {
		allFieldName[rule.Expression.Field.Name] = true
		for _, s := range rule.Expression.Sum {
			allFieldName[s.Name] = true
		}
		for _, n := range appendFieldName {
			newRule := rule
			newRule.Expression.Field.Name += n
			newRule.Expression.Field.DefaultValue = &zeroFloat64
			newSum := make([]Field, len(rule.Expression.Sum))
			for i, s := range rule.Expression.Sum {
				newSum[i] = s
				newSum[i].Name += n
				newSum[i].DefaultValue = &zeroFloat64
				newSum[i].Multiplier = s.Multiplier
				newSum[i].Indicator = s.Indicator
			}
			newRule.Expression.Sum = newSum
			setDefault(&newRule)
			result = append(result, newRule)
		}
	}
	for fieldName := range allFieldName {
		newRule := Rule{}
		newRule.Expression.Field.Name = fieldName + appendFieldName[0]
		newRule.Expression.Field.DefaultValue = &zeroFloat64
		newRule.RuleFlags = []string{"XML_AUTO"}
		newRule.Expression.Sum = append(newRule.Expression.Sum, Field{
			Name:         fieldName + appendFieldName[1],
			DefaultValue: &zeroFloat64,
		})
		newRule.Expression.Sum = append(newRule.Expression.Sum, Field{
			Name:         fieldName + appendFieldName[2],
			DefaultValue: &zeroFloat64,
		})
		setDefault(&newRule)
		result = append(result, newRule)
	}
	return result, nil
}

type RuleProcessor = func([]Rule) ([]Rule, error)

type JsonRuleProcessor struct {
	Rules []Rule
}

func (jrp *JsonRuleProcessor) ProcessRule(location string, ruleProcessor RuleProcessor) error {
	input, err := os.Open(location)
	if err != nil {
		return err
	}
	defer input.Close()
	var rules []Rule
	inputScanner := bufio.NewScanner(input)
	var buffer bytes.Buffer
	for inputScanner.Scan() {
		jsonInput := inputScanner.Bytes()
		if len(jsonInput) == 0 {
			var lineRules []Rule
			jsonData := buffer.Bytes()
			if len(jsonData) == 0 {
				continue
			}
			err = json.NewDecoder(bytes.NewReader(jsonData)).Decode(&lineRules)
			if err != nil {
				return err
			}
			rules = append(rules, lineRules...)
			buffer.Truncate(0)
		} else {
			buffer.Write(jsonInput)
		}
	}
	rules, err = ruleProcessor(rules)
	if err != nil {
		return err
	}
	jrp.Rules = append(jrp.Rules, rules...)
	return nil
}

func (jrp *JsonRuleProcessor) WriteTo(outputFileLocation string) error {
	jsonOutput, err := os.Create(outputFileLocation)
	if err != nil {
		return err
	}
	defer jsonOutput.Close()
	err = json.NewEncoder(jsonOutput).Encode(jrp.Rules)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	flag.Parse()
	jrp := &JsonRuleProcessor{}
	var err error
	err = jrp.ProcessRule(*threeColumnRuleFilepath, threeColumnRule)
	if err != nil {
		panic(err)
	}
	err = jrp.ProcessRule(*normalRuleFilepath, normalRule)
	if err != nil {
		panic(err)
	}
	err = jrp.WriteTo(*outputRuleFilepath)
	if err != nil {
		panic(err)
	}
}

func setDefault(rule *Rule) {
	rule.DecimalPrecision = 2
	if len(rule.ErrorCode) == 0 {
		rule.ErrorCode = "E02PND50XXX"
	}
}
