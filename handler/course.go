package handler

import (
	"fmt"
	"os"
	"playground_backend/common"
	"playground_backend/http"
	"playground_backend/models"
	"strings"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"github.com/pkg/errors"
)

const (
	DEFAULT       = "default"
	CUSTOMIZATION = "customization"
	TRYME         = "tryme"
	CONTAINER     = "container.tmpl"
	VM            = "vm.tmpl"
	LXD           = "lxd/x86.tmpl"
)

type CourseReqParameter struct {
	Token       string                `json:"token"`
	UserId      int64                 `json:"userId"`
	CourseId    string                `json:"courseId"`
	CourseName  string                `json:"courseName"`
	Status      int                   `json:"status"`
	ChapterInfo []ChapterReqParameter `json:"chapterInfo"`
}

type ChapterReqParameter struct {
	ChapterId   string `json:"chapterId"`
	ChapterName string `json:"chapterName"`
	Status      int    `json:"status"`
}

type RspCourse struct {
	CId   int64  `json:"cId"`
	State string `json:"state"`
}

type RspCourseData struct {
	CourseId    string                 `json:"courseId"`
	CourseName  string                 `json:"courseName"`
	Status      int                    `json:"status"`
	IsOnline    int8                   `json:"isOnline"`
	ChapterData []RspCourseChapterData `json:"chapterInfo"`
}

type RspCourseChapterData struct {
	ChapterId   string `json:"chapterId"`
	ChapterName string `json:"chapterName"`
	Status      int    `json:"status"`
	IsOnline    int8   `json:"isOnline"`
}

type EnvPrams struct {
	OnlineEnv        string
	OfflineEnv       string
	CourseUrl        string
	ChapterUrl       string
	ChapterDetailUrl string
}

type EulerBranch struct {
	imageid string
}

type ChapterDetailData struct {
	Title       string
	Description string
	Details     interface{}
	Environment interface{}
	Backend     EulerBranch
}

func addUserCourse(crp CourseReqParameter, uc *models.UserCourse, flag int) {
	uc.CourseId = crp.CourseId
	uc.UserId = crp.UserId
	uc.Status = 1
	uc.CourseName = crp.CourseName
	uc.CompletedFlag = crp.Status
	if flag == 1 {
		uc.CreateTime = common.GetCurTime()
	} else {
		uc.UpdateTime = common.GetCurTime()
	}
}

func UserBoundBourse(crp CourseReqParameter) int64 {
	crd := models.Courses{CourseId: crp.CourseId}
	ccp := models.CoursesChapter{CourseId: crp.CourseId, ChapterId: ""}
	WriteCourseData(crp.UserId, "0", crp.CourseId, "", "User bound course",
		"", "Processing", "",
		crp.Status, crp.Status, &crd, &ccp)
	uc := models.UserCourse{UserId: crp.UserId, CourseId: crp.CourseId}
	ucId := int64(0)
	queryErr := models.QueryUserCourse(&uc, "UserId", "CourseId")
	if uc.Id > 0 || queryErr == nil {
		addUserCourse(crp, &uc, 2)
		uc.CId = crd.Id
		upErr := models.UpdateUserCourse(&uc, "CourseName", "CompletedFlag", "UpdateTime", "CId")
		if upErr != nil {
			logs.Error("UserBoundBourse, upErr: ", upErr)
			WriteCourseData(crp.UserId, "0", crp.CourseId, "", "User bound course",
				"", "failed", "User binding course failed",
				crp.Status, crp.Status, &crd, &ccp)
			return 0
		}
		ucId = uc.Id
	} else {
		addUserCourse(crp, &uc, 1)
		uc.CId = crd.Id
		id, inErr := models.InsertUserCourse(&uc)
		if inErr != nil {
			logs.Error("UserBoundBourse, inErr: ", inErr, ",id:", id)
			WriteCourseData(crp.UserId, "0", crp.CourseId, "", "User bound course",
				"", "failed", "User binding course failed",
				crp.Status, crp.Status, &crd, &ccp)
			return 0
		}
		ucId = id
	}
	// Determining whether a course has been completed
	IsCompleteCourse(crp.CourseId, crp.UserId)
	WriteCourseData(crp.UserId, "0", crp.CourseId, "", "User bound course",
		"", "success", "User binding course successfully",
		crp.Status, crp.Status, &crd, &ccp)
	return ucId
}

func IsCompleteCourse(courseId string, userId int64) {
	completeFlag := true
	ccp := models.QueryAllCourseChapterById(courseId)
	if len(ccp) > 0 {
		for _, cp := range ccp {
			ucc := models.UserCourseChapter{UserId: userId, CourseId: courseId, ChapterId: cp.ChapterId, Status: 1}
			qccErr := models.QueryUserCourseChapter(&ucc, "UserId", "CourseId", "ChapterId", "Status")
			if qccErr == nil && ucc.CompletedFlag == 2 {
				logs.Info("The current chapter has been studied, chapter info: ", cp)
				continue
			} else {
				completeFlag = false
			}
		}
		if completeFlag {
			models.UpdateUserCourseCompleted(2, courseId, userId)
		}
	}
}

func addUserCourseChapter(crp ChapterReqParameter, uc *models.UserCourseChapter,
	ucId, userId int64, courseId string) {
	uc.UcId = ucId
	uc.CourseId = courseId
	uc.UserId = userId
	uc.ChapterId = crp.ChapterId
	uc.ChapterName = crp.ChapterName
	uc.CompletedFlag = crp.Status
	uc.Status = 1
	uc.CreateTime = common.GetCurTime()
	uc.UpdateTime = common.GetCurTime()
}

func UserBoundBourseChapter(crp ChapterReqParameter, ucId, userId int64, courseId string, courseStatus int) {
	crd := models.Courses{CourseId: courseId}
	ccp := models.CoursesChapter{CourseId: courseId, ChapterId: crp.ChapterId}
	WriteCourseData(userId, "0", courseId, crp.ChapterId, "User bound chapter",
		"", "Processing", "",
		courseStatus, crp.Status, &crd, &ccp)
	uc := models.UserCourseChapter{UserId: userId, CourseId: courseId, ChapterId: crp.ChapterId}
	queryErr := models.QueryUserCourseChapter(&uc, "UserId", "CourseId", "ChapterId")
	if uc.Id > 0 || queryErr == nil {
		uc.CId = crd.Id
		uc.TId = ccp.Id
		uc.CourseName = crd.Title
		addUserCourseChapter(crp, &uc, ucId, userId, courseId)
		upErr := models.UpdateUserCourseChapter(&uc, "UcId", "ChapterName",
			"CompletedFlag", "UpdateTime", "CId", "TId", "CourseName")
		if upErr != nil {
			logs.Error("UserBoundBourseChapter, upErr: ", upErr)
			WriteCourseData(userId, "0", courseId, crp.ChapterId, "User bound chapter",
				"", "failed", "User binding course chapter failed",
				courseStatus, crp.Status, &crd, &ccp)
			return
		}
	} else {
		uc.CId = crd.Id
		uc.TId = ccp.Id
		uc.CourseName = crd.Title
		addUserCourseChapter(crp, &uc, ucId, userId, courseId)
		id, inErr := models.InsertUserCourseChapter(&uc)
		if inErr != nil {
			logs.Error("UserBoundBourseChapter, inErr: ", inErr, ",id:", id)
			WriteCourseData(userId, "0", courseId, crp.ChapterId, "User bound chapter",
				"", "failed", "User binding course chapter failed",
				courseStatus, crp.Status, &crd, &ccp)
			return
		}
	}
	WriteCourseData(userId, "0", courseId, crp.ChapterId, "User bound chapter",
		"", "success", "User binding course chapter successfully",
		courseStatus, crp.Status, &crd, &ccp)
}

func (ep EnvPrams) AddCourseToDb(cr interface{}) {
	cor := cr.(map[string]interface{})
	statusList := cor["status"].([]interface{})
	onEnvList := strings.Split(ep.OnlineEnv, ",")
	if len(statusList) > 0 && len(onEnvList) > 0 {
		for _, st := range statusList {
			status := st.(string)
			status = strings.ToLower(status)
			if status == ep.OfflineEnv {
				cr := models.Courses{}
				cr.CourseId = cor["id"].(string)
				queryErr := models.QueryCourse(&cr, "CourseId")
				if cr.Id > 0 {
					cr.Status = 2
					cr.DeleteTime = common.GetCurTime()
					cr.Flag = 1
					delErr := models.UpdateCourse(&cr, "Status", "DeleteTime", "Flag")
					if delErr == nil {
						delChapterErr := models.UpdateCourseAllChapter(cr.Status, 1, cr.CourseId)
						if delChapterErr != nil {
							logs.Error("AddCourseToDb, delChapterErr: ", delChapterErr)
						}
					}
				} else {
					logs.Info("AddCourseToDb, The course does not exist, "+
						"no need to go offline, queryErr: ", queryErr)
				}
			}
			for _, env := range onEnvList {
				if status == env {
					imageid := ""
					coursePathName := cor["content_dir"].(string)
					courseId := cor["id"].(string)
					chapterUrl := fmt.Sprintf(ep.ChapterUrl, coursePathName)
					body, resErr := http.HTTPGitGet(chapterUrl)
					logs.Error(body, "----------------:chapterUrl:", chapterUrl)
					if resErr != nil {
						logs.Error("AddCourseToDb, resErr: ", resErr, ",body: ", body)
						continue
					}
					cId := int64(0)
					cr := models.Courses{}
					cr.CourseId = courseId
					queryErr := models.QueryCourse(&cr, "CourseId")
					if cr.Id > 0 {
						cr.Name = cor["content_dir"].(string)
						AddCourseData(body, &cr)
						upErr := models.UpdateCourse(&cr, "Status", "Name",
							"Title", "Description", "Icon", "Poster", "Banner",
							"Estimated", "UpdateTime", "Flag")
						if upErr != nil {
							logs.Error("AddCourseToDb, upErr: ", upErr)
						}
						cId = cr.Id
					} else {
						cr.CourseId = courseId
						cr.Name = cor["content_dir"].(string)
						AddCourseData(body, &cr)
						id, inErr := models.InsertCourse(&cr)
						if inErr != nil {
							logs.Error("AddCourseToDb, inErr: ", inErr, ",queryErr: ", queryErr)
						}
						cId = id
					}
					upChapterErr := models.UpdateCourseAllChapter(2, 2, cr.CourseId)
					if upChapterErr != nil {
						logs.Error("AddCourseToDb, delChapterErr: ", upChapterErr)
					}
					chapterList := body["chapters"].([]interface{})
					if len(chapterList) > 0 {
						for _, ct := range chapterList {
							chapter := ct.(map[string]interface{})
							cdd := ChapterDetailData{}
							chapterId := chapter["content_dir"].(string)
							ep.GetChapterDetail(coursePathName, chapterId, &cdd)
							imageid = cdd.Backend.imageid
							cp := models.CoursesChapter{CourseId: cr.CourseId, ChapterId: chapterId, EulerBranch: cdd.Backend.imageid}
							querychapterErr := models.QueryCourseChapter(&cp, "CourseId", "ChapterId")
							if cp.Id > 0 {
								AddChapterData(chapter, &cp, cId)
								cp.EulerBranch = cdd.Backend.imageid
								upChapterErr := models.UpdateCourseChapter(&cp,
									"Status", "Title", "Description", "Estimated", "UpdateTime", "EulerBranch")
								if upChapterErr != nil {
									logs.Error("UpdateCourseChapter, upChapterErr: ", upChapterErr)
								}
							} else {
								cp.CourseId = cr.CourseId
								cp.EulerBranch = cdd.Backend.imageid
								AddChapterData(chapter, &cp, cId)
								_, inChapterErr := models.InsertCourseChapter(&cp)
								if inChapterErr != nil {
									logs.Error("InsertCourseChapter, inChapterErr: ",
										inChapterErr, ",querychapterErr: ", querychapterErr)
								}
							}
						}
					}
					if len(imageid) > 1 {
						upErr := models.UpdateCourseByCId(courseId, imageid)
						if upErr != nil {
							logs.Error("UpdateCourseByCId,upErr: ", upErr, courseId, imageid)
						}
						ProcCourseAndResRel(courseId, coursePathName, imageid)
					}
				}
			}
		}
	}
}

func ProcCourseAndResRel(courseId, courseDir, eulerBranch string) {
	rcp := models.ResourceConfigPath{EulerBranch: eulerBranch}
	rr := ReqResource{CourseId: courseId}
	rcpErr := rr.SaveCourseAndResRel(&rcp, courseDir)
	if rcpErr != nil {
		logs.Error("ProcCourseAndResRel, rcpErr: ", rcpErr)
		return
	}
}

func (rr *ReqResource) SaveCourseAndResRel(rcp *models.ResourceConfigPath, courseDir string) error {
	//trymePath := fmt.Sprintf("%v/%v.tmpl", TRYME, rcp.EulerBranch)
	originTemplatePath := fmt.Sprintf("%v", rcp.EulerBranch)
	defTemplatePath := fmt.Sprintf("%v/%v", DEFAULT, rcp.EulerBranch)
	defContainerTemplatePath := fmt.Sprintf("%v/%v_%v", DEFAULT, rcp.EulerBranch, CONTAINER)
	customTemplatePath := fmt.Sprintf("%v/%v_%v_%v", CUSTOMIZATION, rcp.EulerBranch, courseDir, CONTAINER)
	oldTemplatePath := fmt.Sprintf("%v/%v", rcp.EulerBranch, LXD)
	//rcp.ResourcePath = trymePath
	//logs.Info("试  :", rcp.ResourcePath)
	//rcpErr := models.QueryResourceConfigPath(rcp, "EulerBranch", "ResourcePath")
	//if rcp.Id > 0 {
	//	rr.EnvResource = rcp.ResourcePath
	//	rr.ResourceId = rcp.ResourceId
	//	saveErr := SaveResourceTemplate(rr)
	//	return saveErr
	//}
	rcp.ResourcePath = originTemplatePath
	logs.Info("试  :", rcp.ResourcePath)
	rcpErr := models.QueryResourceConfigPath(rcp, "EulerBranch", "ResourcePath")
	if rcp.Id > 0 {
		rr.EnvResource = rcp.ResourcePath
		rr.ResourceId = rcp.ResourceId
		saveErr := SaveResourceTemplate(rr)
		return saveErr
	}
	rcp.ResourcePath = defTemplatePath
	logs.Info("试  :", rcp.ResourcePath)
	rcpErr = models.QueryResourceConfigPath(rcp, "EulerBranch", "ResourcePath")
	if rcp.Id > 0 {
		rr.EnvResource = rcp.ResourcePath
		rr.ResourceId = rcp.ResourceId
		saveErr := SaveResourceTemplate(rr)
		return saveErr
	}

	rcp.ResourcePath = defContainerTemplatePath
	logs.Info("试  :", rcp.ResourcePath)
	rcpErr = models.QueryResourceConfigPath(rcp, "EulerBranch", "ResourcePath")
	if rcp.Id > 0 {
		rr.EnvResource = rcp.ResourcePath
		rr.ResourceId = rcp.ResourceId
		saveErr := SaveResourceTemplate(rr)
		return saveErr
	}
	rcp.ResourcePath = customTemplatePath
	logs.Info("试  :", rcp.ResourcePath)
	rcpErr = models.QueryResourceConfigPath(rcp, "EulerBranch", "ResourcePath")
	if rcp.Id > 0 {
		rr.EnvResource = rcp.ResourcePath
		rr.ResourceId = rcp.ResourceId
		saveErr := SaveResourceTemplate(rr)
		return saveErr
	}

	rcp.ResourcePath = oldTemplatePath
	logs.Info("试  :", rcp.ResourcePath)
	rcpErr = models.QueryResourceConfigPath(rcp, "EulerBranch", "ResourcePath")
	if rcp.Id > 0 {
		rr.EnvResource = rcp.ResourcePath
		rr.ResourceId = rcp.ResourceId
		saveErr := SaveResourceTemplate(rr)
		return saveErr
	}
	if rcpErr != nil {
		logs.Error("QueryResourceConfigPath, rcpErr: ", rcpErr)
	}
	return rcpErr
}

func (ep EnvPrams) GetChapterDetail(coursePathName, chapterId string, cdd *ChapterDetailData) {
	chapterDetailUrl := fmt.Sprintf(ep.ChapterDetailUrl, coursePathName, chapterId)
	body, resErr := http.HTTPGitGet(chapterDetailUrl)
	if resErr != nil {
		logs.Error("AddCourseToDb, resErr: ", resErr, ",body: ", body)
		return
	}
	cdd.Title = body["title"].(string)
	cdd.Description = body["description"].(string)
	eulerBranch := body["backend"].(map[string]interface{})
	imageid, ok := eulerBranch["image_id"]
	if ok {
		cdd.Backend.imageid = imageid.(string)
	}
}

func AddChapterData(cor map[string]interface{}, cr *models.CoursesChapter, cId int64) {
	cr.Status = 1
	cr.CId = cId
	cr.ChapterId = cor["content_dir"].(string)
	cr.Title = cor["title"].(string)
	cr.Description = cor["description"].(string)
	cr.Estimated = cor["estimated_time"].(string)
	cr.UpdateTime = common.GetCurTime()
	cr.CreateTime = common.GetCurTime()
}

func AddCourseData(cor map[string]interface{}, cr *models.Courses) {
	cr.Status = 1
	cr.Flag = 1
	cr.Title = cor["title"].(string)
	cr.Description = cor["description"].(string)
	cr.Icon = cor["logo"].(string)
	cr.Poster = cor["poster"].(string)
	cr.Banner = cor["cover"].(string)
	cr.Estimated = cor["container_live_time"].(string)
	cr.UpdateTime = common.GetCurTime()
	cr.CreateTime = common.GetCurTime()
}

func (ep EnvPrams) ParsingCourse(body map[string]interface{}) error {
	courses, ok := body["courses"]
	if !ok {
		logs.Error("The course list file is abnormal and cannot be parsed")
		return errors.New("The course list file is abnormal and cannot be parsed")
	}
	cousrseList := courses.([]interface{})
	if len(cousrseList) > 0 {
		models.UpdateCourseFlag(2)
		for _, cousrse := range cousrseList {
			ep.AddCourseToDb(cousrse)
		}
		courseList := models.QueryAllCourseData(0)
		if len(courseList) > 0 {
			for _, cs := range courseList {
				if cs.Flag == 2 {
					cs.Status = 2
					cs.DeleteTime = common.GetCurTime()
					models.UpdateCourse(&cs, "Status", "DeleteTime")
					models.UpdateCourseAllChapter(2, 1, cs.CourseId)
				}
			}
		}
		return nil
	}
	logs.Error("ParsingCourse, Course list is empty")
	return errors.New("ParsingCourse, Course list is empty")
}

func SyncCourse() error {
	onlineEnv := beego.AppConfig.String("courses::online_env")
	offlineEnv := beego.AppConfig.String("courses::offline_env")
	courseUrl := beego.AppConfig.String("courses::course_url")
	chapterUrl := beego.AppConfig.String("courses::chapter_url")
	chapterDetailUrl := beego.AppConfig.String("courses::chapter_detail_url")
	if os.Getenv("COURSE_URL") != "" {
		courseUrl = os.Getenv("COURSE_URL")
	}
	if os.Getenv("CHAPTER_URL") != "" {
		chapterUrl = os.Getenv("CHAPTER_URL")
	}
	if os.Getenv("CHAPTER_DETAIL_URL") != "" {
		chapterDetailUrl = os.Getenv("CHAPTER_DETAIL_URL")
	}
	logs.Error("---------------:courseUrl:", courseUrl)
	body, resErr := http.HTTPGitGet(courseUrl)
	if resErr != nil {
		logs.Error("SyncCourse, resErr: ", resErr)
		return resErr
	}
	ep := EnvPrams{OnlineEnv: onlineEnv, OfflineEnv: offlineEnv, CourseUrl: courseUrl,
		ChapterUrl: chapterUrl, ChapterDetailUrl: chapterDetailUrl}
	pErr := ep.ParsingCourse(body)
	if pErr != nil {
		logs.Error("pErr: ", pErr)
	}
	// Operations related to clearing offline courses
	CleanUpCoursePool()
	return pErr
}

func CleanUpCoursePool() {
	courseData := models.QueryAllCourseData(2)
	if len(courseData) > 0 {
		for _, cs := range courseData {
			// 1. Clear environment configuration
			rtr := models.ResourceTempathRel{CourseId: cs.CourseId}
			delErr := models.DeleteResourceTempathRel(&rtr, "CourseId")
			if delErr != nil {
				logs.Error("delErr: ", delErr)
			}
			// 2. Clear User Courses
			upErr := models.UpdateUserCourseByCourseId(2, cs.CourseId)
			if upErr == nil {
				upErr = models.UpdateUserCourseChapterByCourseId(2, cs.CourseId)
				if upErr != nil {
					logs.Error("upErr: ", upErr)
				}
			}
		}
	}
	// 1. Clear User Chapters
	chapterData := models.QueryAllCourseChapterData(2)
	if len(chapterData) > 0 {
		for _, cd := range chapterData {
			upErr := models.UpdateUserCourseChapterByChapterId(2, cd.CourseId, cd.ChapterId)
			if upErr != nil {
				logs.Error("upErr: ", upErr)
			}
		}
	}
}

func SyncCourseData() {
	syncErr := SyncCourse()
	if syncErr != nil {
		logs.Error("syncErr: ", syncErr)
	}
	courseData := models.QueryAllCourseData(1)
	if len(courseData) > 0 {
		for _, csd := range courseData {
			ProcCourseAndResRel(csd.CourseId, csd.Name, csd.EulerBranch)
		}
	}
}

func GetUserCourse(userId int64, currentPage, pageSize int) (rcdList []RspCourseData) {
	// Inquire about course details
	ucList := models.QueryUserCourseData(currentPage, pageSize, userId)
	if len(ucList) > 0 {
		for _, uc := range ucList {
			// Determining whether a course has been completed
			IsCompleteCourse(uc.CourseId, userId)
			crd := models.Courses{CourseId: uc.CourseId}
			ccp := models.CoursesChapter{CourseId: uc.CourseId}
			WriteCourseData(userId, "0", uc.CourseId, "", "Query user's courses",
				"", "success", "Query the user has taken courses successfully",
				uc.CompletedFlag, uc.CompletedFlag, &crd, &ccp)
			rcd := RspCourseData{}
			rccdList := []RspCourseChapterData{}
			cour := models.Courses{CourseId: uc.CourseId}
			quyCourErr := models.QueryCourse(&cour, "CourseId")
			if quyCourErr == nil {
				uc.CourseName = cour.Title
				uc.Status = cour.Status
				ucpList := models.QueryChapterByCourseId(uc.CourseId, userId)
				if len(ucpList) > 0 {
					for _, ucp := range ucpList {
						ccp.ChapterId = ucp.ChapterId
						WriteCourseData(userId, "0", uc.CourseId, ucp.ChapterId, "Query user's courses chapters",
							"", "success", "Query the user's learned course chapters successfully",
							uc.CompletedFlag, ucp.CompletedFlag, &crd, &ccp)
						ccp := models.CoursesChapter{CourseId: ucp.CourseId, ChapterId: ucp.ChapterId}
						quyChapterErr := models.QueryCourseChapter(&ccp, "CourseId", "ChapterId")
						if quyChapterErr == nil {
							ucp.ChapterName = ccp.Title
							ucp.Status = ccp.Status
							rccd := RspCourseChapterData{}
							RspChapter(ucp, &rccd)
							rccdList = append(rccdList, rccd)
						}
					}
				}
			}
			AddRspCourse(uc, &rcd)
			rcd.ChapterData = rccdList
			rcdList = append(rcdList, rcd)
		}
	}
	return
}

func AddRspCourse(uc models.UserCourse, rcd *RspCourseData) {
	rcd.IsOnline = uc.Status
	rcd.Status = uc.CompletedFlag
	rcd.CourseId = uc.CourseId
	rcd.CourseName = uc.CourseName
}

func RspChapter(ucp models.UserCourseChapter, rccd *RspCourseChapterData) {
	rccd.Status = ucp.CompletedFlag
	rccd.ChapterName = ucp.ChapterName
	rccd.ChapterId = ucp.ChapterId
	rccd.IsOnline = ucp.Status
}
