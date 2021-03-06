/*
Real-time Charging System for Telecom & ISP environments
Copyright (C) 2012-2015 ITsysCOM GmbH

This program is free software: you can Storagetribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, ornt
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITH*out ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>
*/

package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cgrates/cgrates/utils"
)

func NewStoredCdrFromExternalCdr(extCdr *ExternalCdr) (*StoredCdr, error) {
	var err error
	storedCdr := &StoredCdr{CgrId: extCdr.CgrId, OrderId: extCdr.OrderId, TOR: extCdr.TOR, AccId: extCdr.AccId, CdrHost: extCdr.CdrHost, CdrSource: extCdr.CdrSource,
		ReqType: extCdr.ReqType, Direction: extCdr.Direction, Tenant: extCdr.Tenant, Category: extCdr.Category, Account: extCdr.Account, Subject: extCdr.Subject,
		Destination: extCdr.Destination, Supplier: extCdr.Supplier, DisconnectCause: extCdr.DisconnectCause, ExtraFields: extCdr.ExtraFields,
		MediationRunId: extCdr.MediationRunId, RatedAccount: extCdr.RatedAccount, RatedSubject: extCdr.RatedSubject, Cost: extCdr.Cost, Rated: extCdr.Rated}
	if storedCdr.SetupTime, err = utils.ParseTimeDetectLayout(extCdr.SetupTime); err != nil {
		return nil, err
	}
	if storedCdr.AnswerTime, err = utils.ParseTimeDetectLayout(extCdr.AnswerTime); err != nil {
		return nil, err
	}
	if storedCdr.Usage, err = utils.ParseDurationWithSecs(extCdr.Usage); err != nil {
		return nil, err
	}
	if len(extCdr.CostDetails) != 0 {
		if err = json.Unmarshal([]byte(extCdr.CostDetails), storedCdr.CostDetails); err != nil {
			return nil, err
		}
	}
	return storedCdr, nil
}

// Kinda standard of internal CDR, complies to CDR interface also
type StoredCdr struct {
	CgrId           string
	OrderId         int64             // Stor order id used as export order id
	TOR             string            // type of record, meta-field, should map to one of the TORs hardcoded inside the server <*voice|*data|*sms>
	AccId           string            // represents the unique accounting id given by the telecom switch generating the CDR
	CdrHost         string            // represents the IP address of the host generating the CDR (automatically populated by the server)
	CdrSource       string            // formally identifies the source of the CDR (free form field)
	ReqType         string            // matching the supported request types by the **CGRateS**, accepted values are hardcoded in the server <prepaid|postpaid|pseudoprepaid|rated>.
	Direction       string            // matching the supported direction identifiers of the CGRateS <*out>
	Tenant          string            // tenant whom this record belongs
	Category        string            // free-form filter for this record, matching the category defined in rating profiles.
	Account         string            // account id (accounting subsystem) the record should be attached to
	Subject         string            // rating subject (rating subsystem) this record should be attached to
	Destination     string            // destination to be charged
	SetupTime       time.Time         // set-up time of the event. Supported formats: datetime RFC3339 compatible, SQL datetime (eg: MySQL), unix timestamp.
	AnswerTime      time.Time         // answer time of the event. Supported formats: datetime RFC3339 compatible, SQL datetime (eg: MySQL), unix timestamp.
	Usage           time.Duration     // event usage information (eg: in case of tor=*voice this will represent the total duration of a call)
	Supplier        string            // Supplier information when available
	DisconnectCause string            // Disconnect cause of the event
	ExtraFields     map[string]string // Extra fields to be stored in CDR
	MediationRunId  string
	RatedAccount    string // Populated out of rating data
	RatedSubject    string
	Cost            float64
	ExtraInfo       string    // Container for extra information related to this CDR, eg: populated with error reason in case of error on calculation
	CostDetails     *CallCost // Attach the cost details to CDR when possible
	Rated           bool      // Mark the CDR as rated so we do not process it during mediation
}

func (storedCdr *StoredCdr) CostDetailsJson() string {
	if storedCdr.CostDetails == nil {
		return ""
	}
	mrshled, _ := json.Marshal(storedCdr.CostDetails)
	return string(mrshled)
}

// Used to multiply usage on export
func (storedCdr *StoredCdr) UsageMultiply(multiplyFactor float64, roundDecimals int) {
	storedCdr.Usage = time.Duration(int(utils.Round(float64(storedCdr.Usage.Nanoseconds())*multiplyFactor, roundDecimals, utils.ROUNDING_MIDDLE))) // Rounding down could introduce a slight loss here but only at nanoseconds level
}

// Used to multiply cost on export
func (storedCdr *StoredCdr) CostMultiply(multiplyFactor float64, roundDecimals int) {
	storedCdr.Cost = utils.Round(storedCdr.Cost*multiplyFactor, roundDecimals, utils.ROUNDING_MIDDLE)
}

// Format cost as string on export
func (storedCdr *StoredCdr) FormatCost(shiftDecimals, roundDecimals int) string {
	cost := storedCdr.Cost
	if shiftDecimals != 0 {
		cost = cost * math.Pow10(shiftDecimals)
	}
	return strconv.FormatFloat(cost, 'f', roundDecimals, 64)
}

// Formats usage on export
func (storedCdr *StoredCdr) FormatUsage(layout string) string {
	if utils.IsSliceMember([]string{utils.DATA, utils.SMS}, storedCdr.TOR) {
		return strconv.FormatFloat(utils.Round(storedCdr.Usage.Seconds(), 0, utils.ROUNDING_MIDDLE), 'f', -1, 64)
	}
	switch layout {
	default:
		return strconv.FormatFloat(float64(storedCdr.Usage.Nanoseconds())/1000000000, 'f', -1, 64)
	}
}

// Used to retrieve fields as string, primary fields are const labeled
func (storedCdr *StoredCdr) FieldAsString(rsrFld *utils.RSRField) string {
	switch rsrFld.Id {
	case utils.CGRID:
		return rsrFld.ParseValue(storedCdr.CgrId)
	case utils.ORDERID:
		return rsrFld.ParseValue(strconv.FormatInt(storedCdr.OrderId, 10))
	case utils.TOR:
		return rsrFld.ParseValue(storedCdr.TOR)
	case utils.ACCID:
		return rsrFld.ParseValue(storedCdr.AccId)
	case utils.CDRHOST:
		return rsrFld.ParseValue(storedCdr.CdrHost)
	case utils.CDRSOURCE:
		return rsrFld.ParseValue(storedCdr.CdrSource)
	case utils.REQTYPE:
		return rsrFld.ParseValue(storedCdr.ReqType)
	case utils.DIRECTION:
		return rsrFld.ParseValue(storedCdr.Direction)
	case utils.TENANT:
		return rsrFld.ParseValue(storedCdr.Tenant)
	case utils.CATEGORY:
		return rsrFld.ParseValue(storedCdr.Category)
	case utils.ACCOUNT:
		return rsrFld.ParseValue(storedCdr.Account)
	case utils.SUBJECT:
		return rsrFld.ParseValue(storedCdr.Subject)
	case utils.DESTINATION:
		return rsrFld.ParseValue(storedCdr.Destination)
	case utils.SETUP_TIME:
		return rsrFld.ParseValue(storedCdr.SetupTime.Format(time.RFC3339))
	case utils.ANSWER_TIME:
		return rsrFld.ParseValue(storedCdr.AnswerTime.Format(time.RFC3339))
	case utils.USAGE:
		return strconv.FormatFloat(utils.Round(storedCdr.Usage.Seconds(), 0, utils.ROUNDING_MIDDLE), 'f', -1, 64)
	case utils.SUPPLIER:
		return rsrFld.ParseValue(storedCdr.Supplier)
	case utils.DISCONNECT_CAUSE:
		return rsrFld.ParseValue(storedCdr.DisconnectCause)
	case utils.MEDI_RUNID:
		return rsrFld.ParseValue(storedCdr.MediationRunId)
	case utils.RATED_ACCOUNT:
		return rsrFld.ParseValue(storedCdr.RatedAccount)
	case utils.RATED_SUBJECT:
		return rsrFld.ParseValue(storedCdr.RatedSubject)
	case utils.COST:
		return rsrFld.ParseValue(strconv.FormatFloat(storedCdr.Cost, 'f', -1, 64)) // Recommended to use FormatCost
	case utils.COST_DETAILS:
		return rsrFld.ParseValue(storedCdr.CostDetailsJson())
	default:
		return rsrFld.ParseValue(storedCdr.ExtraFields[rsrFld.Id])
	}
}

func (storedCdr *StoredCdr) PassesFieldFilter(fieldFilter *utils.RSRField) (bool, string) {
	if fieldFilter == nil {
		return true, ""
	}
	if fieldFilter.IsStatic() && storedCdr.FieldAsString(&utils.RSRField{Id: fieldFilter.Id}) == storedCdr.FieldAsString(fieldFilter) {
		return true, storedCdr.FieldAsString(&utils.RSRField{Id: fieldFilter.Id})
	}
	preparedFilter := &utils.RSRField{Id: fieldFilter.Id, RSRules: make([]*utils.ReSearchReplace, len(fieldFilter.RSRules))} // Reset rules so they do not point towards same structures as original fieldFilter
	for idx := range fieldFilter.RSRules {
		// Hardcode the template with maximum of 5 groups ordered
		preparedFilter.RSRules[idx] = &utils.ReSearchReplace{SearchRegexp: fieldFilter.RSRules[idx].SearchRegexp, ReplaceTemplate: utils.FILTER_REGEXP_TPL}
	}
	preparedVal := storedCdr.FieldAsString(preparedFilter)
	filteredValue := storedCdr.FieldAsString(fieldFilter)
	if preparedFilter.RegexpMatched() && (len(preparedVal) == 0 || preparedVal == filteredValue) {
		return true, filteredValue
	}
	return false, ""
}

func (storedCdr *StoredCdr) AsStoredCdr() *StoredCdr {
	return storedCdr
}

// Ability to send the CgrCdr remotely to another CDR server, we do not include rating variables for now
func (storedCdr *StoredCdr) AsHttpForm() url.Values {
	v := url.Values{}
	for fld, val := range storedCdr.ExtraFields {
		v.Set(fld, val)
	}
	v.Set(utils.TOR, storedCdr.TOR)
	v.Set(utils.ACCID, storedCdr.AccId)
	v.Set(utils.CDRHOST, storedCdr.CdrHost)
	v.Set(utils.CDRSOURCE, storedCdr.CdrSource)
	v.Set(utils.REQTYPE, storedCdr.ReqType)
	v.Set(utils.DIRECTION, storedCdr.Direction)
	v.Set(utils.TENANT, storedCdr.Tenant)
	v.Set(utils.CATEGORY, storedCdr.Category)
	v.Set(utils.ACCOUNT, storedCdr.Account)
	v.Set(utils.SUBJECT, storedCdr.Subject)
	v.Set(utils.DESTINATION, storedCdr.Destination)
	v.Set(utils.SETUP_TIME, storedCdr.SetupTime.Format(time.RFC3339))
	v.Set(utils.ANSWER_TIME, storedCdr.AnswerTime.Format(time.RFC3339))
	v.Set(utils.USAGE, storedCdr.FormatUsage(utils.SECONDS))
	v.Set(utils.SUPPLIER, storedCdr.Supplier)
	v.Set(utils.DISCONNECT_CAUSE, storedCdr.DisconnectCause)
	if storedCdr.CostDetails != nil {
		v.Set(utils.COST_DETAILS, storedCdr.CostDetailsJson())
	}
	return v
}

// Used in mediation, primaryMandatory marks whether missing field out of request represents error or can be ignored
func (storedCdr *StoredCdr) ForkCdr(runId string, reqTypeFld, directionFld, tenantFld, categFld, accountFld, subjectFld, destFld, setupTimeFld,
	answerTimeFld, durationFld, supplierFld, disconnectCauseFld *utils.RSRField,
	extraFlds []*utils.RSRField, primaryMandatory bool) (*StoredCdr, error) {
	if reqTypeFld == nil {
		reqTypeFld, _ = utils.NewRSRField(utils.META_DEFAULT)
	}
	if reqTypeFld.Id == utils.META_DEFAULT {
		reqTypeFld.Id = utils.REQTYPE
	}
	if directionFld == nil {
		directionFld, _ = utils.NewRSRField(utils.META_DEFAULT)
	}
	if directionFld.Id == utils.META_DEFAULT {
		directionFld.Id = utils.DIRECTION
	}
	if tenantFld == nil {
		tenantFld, _ = utils.NewRSRField(utils.META_DEFAULT)
	}
	if tenantFld.Id == utils.META_DEFAULT {
		tenantFld.Id = utils.TENANT
	}
	if categFld == nil {
		categFld, _ = utils.NewRSRField(utils.META_DEFAULT)
	}
	if categFld.Id == utils.META_DEFAULT {
		categFld.Id = utils.CATEGORY
	}
	if accountFld == nil {
		accountFld, _ = utils.NewRSRField(utils.META_DEFAULT)
	}
	if accountFld.Id == utils.META_DEFAULT {
		accountFld.Id = utils.ACCOUNT
	}
	if subjectFld == nil {
		subjectFld, _ = utils.NewRSRField(utils.META_DEFAULT)
	}
	if subjectFld.Id == utils.META_DEFAULT {
		subjectFld.Id = utils.SUBJECT
	}
	if destFld == nil {
		destFld, _ = utils.NewRSRField(utils.META_DEFAULT)
	}
	if destFld.Id == utils.META_DEFAULT {
		destFld.Id = utils.DESTINATION
	}
	if setupTimeFld == nil {
		setupTimeFld, _ = utils.NewRSRField(utils.META_DEFAULT)
	}
	if setupTimeFld.Id == utils.META_DEFAULT {
		setupTimeFld.Id = utils.SETUP_TIME
	}
	if answerTimeFld == nil {
		answerTimeFld, _ = utils.NewRSRField(utils.META_DEFAULT)
	}
	if answerTimeFld.Id == utils.META_DEFAULT {
		answerTimeFld.Id = utils.ANSWER_TIME
	}
	if durationFld == nil {
		durationFld, _ = utils.NewRSRField(utils.META_DEFAULT)
	}
	if durationFld.Id == utils.META_DEFAULT {
		durationFld.Id = utils.USAGE
	}
	if supplierFld == nil {
		supplierFld, _ = utils.NewRSRField(utils.META_DEFAULT)
	}
	if supplierFld.Id == utils.META_DEFAULT {
		supplierFld.Id = utils.SUPPLIER
	}
	if disconnectCauseFld == nil {
		disconnectCauseFld, _ = utils.NewRSRField(utils.META_DEFAULT)
	}
	if disconnectCauseFld.Id == utils.META_DEFAULT {
		disconnectCauseFld.Id = utils.DISCONNECT_CAUSE
	}
	var err error
	frkStorCdr := new(StoredCdr)
	frkStorCdr.CgrId = storedCdr.CgrId
	frkStorCdr.TOR = storedCdr.TOR
	frkStorCdr.MediationRunId = runId
	frkStorCdr.Cost = -1.0 // Default for non-rated CDR
	frkStorCdr.AccId = storedCdr.AccId
	frkStorCdr.CdrHost = storedCdr.CdrHost
	frkStorCdr.CdrSource = storedCdr.CdrSource
	frkStorCdr.ReqType = storedCdr.FieldAsString(reqTypeFld)
	if primaryMandatory && len(frkStorCdr.ReqType) == 0 {
		return nil, errors.New(fmt.Sprintf("%s:%s:%s", utils.ERR_MANDATORY_IE_MISSING, utils.REQTYPE, reqTypeFld.Id))
	}
	frkStorCdr.Direction = storedCdr.FieldAsString(directionFld)
	if primaryMandatory && len(frkStorCdr.Direction) == 0 {
		return nil, errors.New(fmt.Sprintf("%s:%s:%s", utils.ERR_MANDATORY_IE_MISSING, utils.DIRECTION, directionFld.Id))
	}
	frkStorCdr.Tenant = storedCdr.FieldAsString(tenantFld)
	if primaryMandatory && len(frkStorCdr.Tenant) == 0 {
		return nil, errors.New(fmt.Sprintf("%s:%s:%s", utils.ERR_MANDATORY_IE_MISSING, utils.TENANT, tenantFld.Id))
	}
	frkStorCdr.Category = storedCdr.FieldAsString(categFld)
	if primaryMandatory && len(frkStorCdr.Category) == 0 {
		return nil, errors.New(fmt.Sprintf("%s:%s:%s", utils.ERR_MANDATORY_IE_MISSING, utils.CATEGORY, categFld.Id))
	}
	frkStorCdr.Account = storedCdr.FieldAsString(accountFld)
	if primaryMandatory && len(frkStorCdr.Account) == 0 {
		return nil, errors.New(fmt.Sprintf("%s:%s:%s", utils.ERR_MANDATORY_IE_MISSING, utils.ACCOUNT, accountFld.Id))
	}
	frkStorCdr.Subject = storedCdr.FieldAsString(subjectFld)
	if primaryMandatory && len(frkStorCdr.Subject) == 0 {
		return nil, errors.New(fmt.Sprintf("%s:%s:%s", utils.ERR_MANDATORY_IE_MISSING, utils.SUBJECT, subjectFld.Id))
	}
	frkStorCdr.Destination = storedCdr.FieldAsString(destFld)
	if primaryMandatory && len(frkStorCdr.Destination) == 0 && frkStorCdr.TOR == utils.VOICE {
		return nil, errors.New(fmt.Sprintf("%s:%s:%s", utils.ERR_MANDATORY_IE_MISSING, utils.DESTINATION, destFld.Id))
	}
	sTimeStr := storedCdr.FieldAsString(setupTimeFld)
	if primaryMandatory && len(sTimeStr) == 0 {
		return nil, errors.New(fmt.Sprintf("%s:%s:%s", utils.ERR_MANDATORY_IE_MISSING, utils.SETUP_TIME, setupTimeFld.Id))
	} else if frkStorCdr.SetupTime, err = utils.ParseTimeDetectLayout(sTimeStr); err != nil {
		return nil, err
	}
	aTimeStr := storedCdr.FieldAsString(answerTimeFld)
	if primaryMandatory && len(aTimeStr) == 0 {
		return nil, errors.New(fmt.Sprintf("%s:%s:%s", utils.ERR_MANDATORY_IE_MISSING, utils.ANSWER_TIME, answerTimeFld.Id))
	} else if frkStorCdr.AnswerTime, err = utils.ParseTimeDetectLayout(aTimeStr); err != nil {
		return nil, err
	}
	durStr := storedCdr.FieldAsString(durationFld)
	if primaryMandatory && len(durStr) == 0 {
		return nil, errors.New(fmt.Sprintf("%s:%s:%s", utils.ERR_MANDATORY_IE_MISSING, utils.USAGE, durationFld.Id))
	} else if frkStorCdr.Usage, err = utils.ParseDurationWithSecs(durStr); err != nil {
		return nil, err
	}
	frkStorCdr.Supplier = storedCdr.FieldAsString(supplierFld)
	frkStorCdr.DisconnectCause = storedCdr.FieldAsString(disconnectCauseFld)
	frkStorCdr.ExtraFields = make(map[string]string, len(extraFlds))
	for _, fld := range extraFlds {
		frkStorCdr.ExtraFields[fld.Id] = storedCdr.FieldAsString(fld)
	}
	return frkStorCdr, nil
}

func (storedCdr *StoredCdr) AsExternalCdr() *ExternalCdr {
	return &ExternalCdr{CgrId: storedCdr.CgrId,
		OrderId:         storedCdr.OrderId,
		TOR:             storedCdr.TOR,
		AccId:           storedCdr.AccId,
		CdrHost:         storedCdr.CdrHost,
		CdrSource:       storedCdr.CdrSource,
		ReqType:         storedCdr.ReqType,
		Direction:       storedCdr.Direction,
		Tenant:          storedCdr.Tenant,
		Category:        storedCdr.Category,
		Account:         storedCdr.Account,
		Subject:         storedCdr.Subject,
		Destination:     storedCdr.Destination,
		SetupTime:       storedCdr.SetupTime.Format(time.RFC3339),
		AnswerTime:      storedCdr.AnswerTime.Format(time.RFC3339),
		Usage:           storedCdr.FormatUsage(utils.SECONDS),
		Supplier:        storedCdr.Supplier,
		DisconnectCause: storedCdr.DisconnectCause,
		ExtraFields:     storedCdr.ExtraFields,
		MediationRunId:  storedCdr.MediationRunId,
		RatedAccount:    storedCdr.RatedAccount,
		RatedSubject:    storedCdr.RatedSubject,
		Cost:            storedCdr.Cost,
		CostDetails:     storedCdr.CostDetailsJson(),
	}
}

// Implementation of Event interface, used in tests
func (storedCdr *StoredCdr) AsEvent(ignored string) Event {
	return Event(storedCdr)
}
func (storedCdr *StoredCdr) GetName() string {
	return storedCdr.CdrSource
}
func (storedCdr *StoredCdr) GetCgrId() string {
	return storedCdr.CgrId
}
func (storedCdr *StoredCdr) GetUUID() string {
	return storedCdr.AccId
}
func (storedCdr *StoredCdr) GetSessionIds() []string {
	return []string{storedCdr.GetUUID()}
}
func (storedCdr *StoredCdr) GetDirection(fieldName string) string {
	if utils.IsSliceMember([]string{utils.DIRECTION, utils.META_DEFAULT}, fieldName) {
		return storedCdr.Direction
	}
	if strings.HasPrefix(fieldName, utils.STATIC_VALUE_PREFIX) { // Static value
		return fieldName[len(utils.STATIC_VALUE_PREFIX):]
	}
	return storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
}
func (storedCdr *StoredCdr) GetSubject(fieldName string) string {
	if utils.IsSliceMember([]string{utils.SUBJECT, utils.META_DEFAULT}, fieldName) {
		return storedCdr.Subject
	}
	if strings.HasPrefix(fieldName, utils.STATIC_VALUE_PREFIX) { // Static value
		return fieldName[len(utils.STATIC_VALUE_PREFIX):]
	}
	return storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
}
func (storedCdr *StoredCdr) GetAccount(fieldName string) string {
	if utils.IsSliceMember([]string{utils.ACCOUNT, utils.META_DEFAULT}, fieldName) {
		return storedCdr.Account
	}
	if strings.HasPrefix(fieldName, utils.STATIC_VALUE_PREFIX) { // Static value
		return fieldName[len(utils.STATIC_VALUE_PREFIX):]
	}
	return storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
}
func (storedCdr *StoredCdr) GetDestination(fieldName string) string {
	if utils.IsSliceMember([]string{utils.DESTINATION, utils.META_DEFAULT}, fieldName) {
		return storedCdr.Destination
	}
	if strings.HasPrefix(fieldName, utils.STATIC_VALUE_PREFIX) { // Static value
		return fieldName[len(utils.STATIC_VALUE_PREFIX):]
	}
	return storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
}
func (storedCdr *StoredCdr) GetCallDestNr(fieldName string) string {
	if utils.IsSliceMember([]string{utils.DESTINATION, utils.META_DEFAULT}, fieldName) {
		return storedCdr.Destination
	}
	if strings.HasPrefix(fieldName, utils.STATIC_VALUE_PREFIX) { // Static value
		return fieldName[len(utils.STATIC_VALUE_PREFIX):]
	}
	return storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
}
func (storedCdr *StoredCdr) GetCategory(fieldName string) string {
	if utils.IsSliceMember([]string{utils.CATEGORY, utils.META_DEFAULT}, fieldName) {
		return storedCdr.Category
	}
	if strings.HasPrefix(fieldName, utils.STATIC_VALUE_PREFIX) { // Static value
		return fieldName[len(utils.STATIC_VALUE_PREFIX):]
	}
	return storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
}
func (storedCdr *StoredCdr) GetTenant(fieldName string) string {
	if utils.IsSliceMember([]string{utils.TENANT, utils.META_DEFAULT}, fieldName) {
		return storedCdr.Tenant
	}
	if strings.HasPrefix(fieldName, utils.STATIC_VALUE_PREFIX) { // Static value
		return fieldName[len(utils.STATIC_VALUE_PREFIX):]
	}
	return storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
}
func (storedCdr *StoredCdr) GetReqType(fieldName string) string {
	if utils.IsSliceMember([]string{utils.REQTYPE, utils.META_DEFAULT}, fieldName) {
		return storedCdr.ReqType
	}
	if strings.HasPrefix(fieldName, utils.STATIC_VALUE_PREFIX) { // Static value
		return fieldName[len(utils.STATIC_VALUE_PREFIX):]
	}
	return storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
}
func (storedCdr *StoredCdr) GetSetupTime(fieldName string) (time.Time, error) {
	if utils.IsSliceMember([]string{utils.SETUP_TIME, utils.META_DEFAULT}, fieldName) {
		return storedCdr.SetupTime, nil
	}
	var sTimeVal string
	if strings.HasPrefix(fieldName, utils.STATIC_VALUE_PREFIX) { // Static value
		sTimeVal = fieldName[len(utils.STATIC_VALUE_PREFIX):]
	} else {
		sTimeVal = storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
	}
	return utils.ParseTimeDetectLayout(sTimeVal)
}
func (storedCdr *StoredCdr) GetAnswerTime(fieldName string) (time.Time, error) {
	if utils.IsSliceMember([]string{utils.ANSWER_TIME, utils.META_DEFAULT}, fieldName) {
		return storedCdr.AnswerTime, nil
	}
	var aTimeVal string
	if strings.HasPrefix(fieldName, utils.STATIC_VALUE_PREFIX) { // Static value
		aTimeVal = fieldName[len(utils.STATIC_VALUE_PREFIX):]
	} else {
		aTimeVal = storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
	}
	return utils.ParseTimeDetectLayout(aTimeVal)
}
func (storedCdr *StoredCdr) GetEndTime() (time.Time, error) {
	return storedCdr.AnswerTime.Add(storedCdr.Usage), nil
}
func (storedCdr *StoredCdr) GetDuration(fieldName string) (time.Duration, error) {
	if utils.IsSliceMember([]string{utils.USAGE, utils.META_DEFAULT}, fieldName) {
		return storedCdr.Usage, nil
	}
	var durVal string
	if strings.HasPrefix(fieldName, utils.STATIC_VALUE_PREFIX) { // Static value
		durVal = fieldName[len(utils.STATIC_VALUE_PREFIX):]
	} else {
		durVal = storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
	}
	return utils.ParseDurationWithSecs(durVal)
}
func (storedCdr *StoredCdr) GetSupplier(fieldName string) string {
	if utils.IsSliceMember([]string{utils.SUPPLIER, utils.META_DEFAULT}, fieldName) {
		return storedCdr.Supplier
	}
	return storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
}
func (storedCdr *StoredCdr) GetDisconnectCause(fieldName string) string {
	if utils.IsSliceMember([]string{utils.DISCONNECT_CAUSE, utils.META_DEFAULT}, fieldName) {
		return storedCdr.DisconnectCause
	}
	return storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
}
func (storedCdr *StoredCdr) GetOriginatorIP(fieldName string) string {
	if utils.IsSliceMember([]string{utils.CDRHOST, utils.META_DEFAULT}, fieldName) {
		return storedCdr.CdrHost
	}
	return storedCdr.FieldAsString(&utils.RSRField{Id: fieldName})
}
func (storedCdr *StoredCdr) GetExtraFields() map[string]string {
	return storedCdr.ExtraFields
}
func (storedCdr *StoredCdr) MissingParameter() bool {
	return len(storedCdr.AccId) == 0 ||
		len(storedCdr.Category) == 0 ||
		len(storedCdr.Tenant) == 0 ||
		len(storedCdr.Account) == 0 ||
		len(storedCdr.Destination) == 0
}
func (storedCdr *StoredCdr) ParseEventValue(rsrFld *utils.RSRField) string {
	return storedCdr.FieldAsString(rsrFld)
}
func (storedCdr *StoredCdr) String() string {
	mrsh, _ := json.Marshal(storedCdr)
	return string(mrsh)
}

type ExternalCdr struct {
	CgrId           string
	OrderId         int64
	TOR             string
	AccId           string
	CdrHost         string
	CdrSource       string
	ReqType         string
	Direction       string
	Tenant          string
	Category        string
	Account         string
	Subject         string
	Destination     string
	SetupTime       string
	AnswerTime      string
	Usage           string
	Supplier        string
	DisconnectCause string
	ExtraFields     map[string]string
	MediationRunId  string
	RatedAccount    string
	RatedSubject    string
	Cost            float64
	CostDetails     string
	Rated           bool // Mark the CDR as rated so we do not process it during mediation
}

// Used when authorizing requests from outside, eg ApierV1.GetMaxSessionTime
type MaxUsageReq struct {
	TOR         string
	ReqType     string
	Direction   string
	Tenant      string
	Category    string
	Account     string
	Subject     string
	Destination string
	SetupTime   string
	AnswerTime  string
	Usage       string
}

func (self *MaxUsageReq) AsStoredCdr() (*StoredCdr, error) {
	var err error
	storedCdr := &StoredCdr{TOR: self.TOR, ReqType: self.ReqType, Direction: self.Direction, Tenant: self.Tenant, Category: self.Category,
		Account: self.Account, Subject: self.Subject, Destination: self.Destination}
	if storedCdr.SetupTime, err = utils.ParseTimeDetectLayout(self.SetupTime); err != nil {
		return nil, err
	}
	if storedCdr.AnswerTime, err = utils.ParseTimeDetectLayout(self.AnswerTime); err != nil {
		return nil, err
	}
	if storedCdr.Usage, err = utils.ParseDurationWithSecs(self.Usage); err != nil {
		return nil, err
	}
	return storedCdr, nil
}
