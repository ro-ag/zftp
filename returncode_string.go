// Code generated by "stringer -type=ReturnCode"; DO NOT EDIT.

package zftp

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[CodeListOK-125]
	_ = x[CodeFileStatusOK-150]
	_ = x[CodeDirStatusOK-151]
	_ = x[CodeCmdOK-200]
	_ = x[CodeCmdNotImplementedSuper-202]
	_ = x[CodeSysStatus-211]
	_ = x[CodeDirStatus-212]
	_ = x[CodeFileStatus-213]
	_ = x[CodeHelpMsg-214]
	_ = x[CodeSysType-215]
	_ = x[CodeSvcReadySoon-220]
	_ = x[CodeSvcClosingControlConn-221]
	_ = x[CodeDataConnOpen-225]
	_ = x[CodeClosingDataConn-226]
	_ = x[CodeEnteringPassiveMode-227]
	_ = x[CodeLoggedInProceed-230]
	_ = x[CodeFileActionOK-250]
	_ = x[CodeDirCreated-257]
	_ = x[CodeNeedPwd-331]
	_ = x[CodeNeedAcctForLogin-332]
	_ = x[CodeSecurityExchangeOK-334]
	_ = x[CodeNeedInfo-350]
	_ = x[CodeSvcNotAvailable-421]
	_ = x[CodeCantOpenDataConn-425]
	_ = x[CodeConnClosed-426]
	_ = x[CodeFileActionNotTaken-450]
	_ = x[CodeLocalError-451]
	_ = x[CodeInsufficientStorage-452]
	_ = x[CodeCmdNotRecognized-500]
	_ = x[CodeArgsError-501]
	_ = x[CodeCmdNotImplemented-502]
	_ = x[CodeBadCmdSequence-503]
	_ = x[CodeCmdNotImplementedParam-504]
	_ = x[CodeUserNotLogged-530]
	_ = x[CodeFileActionNotTakenPerm-550]
	_ = x[CodePageTypeUnknown-551]
	_ = x[CodeExceededStorageAlloc-552]
	_ = x[CodeBadFileName-553]
}

const _ReturnCode_name = "CodeListOKCodeFileStatusOKCodeDirStatusOKCodeCmdOKCodeCmdNotImplementedSuperCodeSysStatusCodeDirStatusCodeFileStatusCodeHelpMsgCodeSysTypeCodeSvcReadySoonCodeSvcClosingControlConnCodeDataConnOpenCodeClosingDataConnCodeEnteringPassiveModeCodeLoggedInProceedCodeFileActionOKCodeDirCreatedCodeNeedPwdCodeNeedAcctForLoginCodeSecurityExchangeOKCodeNeedInfoCodeSvcNotAvailableCodeCantOpenDataConnCodeConnClosedCodeFileActionNotTakenCodeLocalErrorCodeInsufficientStorageCodeCmdNotRecognizedCodeArgsErrorCodeCmdNotImplementedCodeBadCmdSequenceCodeCmdNotImplementedParamCodeUserNotLoggedCodeFileActionNotTakenPermCodePageTypeUnknownCodeExceededStorageAllocCodeBadFileName"

var _ReturnCode_map = map[ReturnCode]string{
	125: _ReturnCode_name[0:10],
	150: _ReturnCode_name[10:26],
	151: _ReturnCode_name[26:41],
	200: _ReturnCode_name[41:50],
	202: _ReturnCode_name[50:76],
	211: _ReturnCode_name[76:89],
	212: _ReturnCode_name[89:102],
	213: _ReturnCode_name[102:116],
	214: _ReturnCode_name[116:127],
	215: _ReturnCode_name[127:138],
	220: _ReturnCode_name[138:154],
	221: _ReturnCode_name[154:179],
	225: _ReturnCode_name[179:195],
	226: _ReturnCode_name[195:214],
	227: _ReturnCode_name[214:237],
	230: _ReturnCode_name[237:256],
	250: _ReturnCode_name[256:272],
	257: _ReturnCode_name[272:286],
	331: _ReturnCode_name[286:297],
	332: _ReturnCode_name[297:317],
	334: _ReturnCode_name[317:339],
	350: _ReturnCode_name[339:351],
	421: _ReturnCode_name[351:370],
	425: _ReturnCode_name[370:390],
	426: _ReturnCode_name[390:404],
	450: _ReturnCode_name[404:426],
	451: _ReturnCode_name[426:440],
	452: _ReturnCode_name[440:463],
	500: _ReturnCode_name[463:483],
	501: _ReturnCode_name[483:496],
	502: _ReturnCode_name[496:517],
	503: _ReturnCode_name[517:535],
	504: _ReturnCode_name[535:561],
	530: _ReturnCode_name[561:578],
	550: _ReturnCode_name[578:604],
	551: _ReturnCode_name[604:623],
	552: _ReturnCode_name[623:647],
	553: _ReturnCode_name[647:662],
}

func (i ReturnCode) String() string {
	if str, ok := _ReturnCode_map[i]; ok {
		return str
	}
	return "ReturnCode(" + strconv.FormatInt(int64(i), 10) + ")"
}
