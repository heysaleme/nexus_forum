package service

import (
	"errors"
	"regexp"
	"strings"
	"sync"
	"time"

	"nexus-forum-backend/internal/model"
	"nexus-forum-backend/internal/repository"
)

type ModerationService interface {
	BanUser(moderatorID, targetUserID uint, reason string) error
	UnbanUser(moderatorID, targetUserID uint, reason string) error
	ShadowBanUser(moderatorID, targetUserID uint, reason string) error
	UnshadowBanUser(moderatorID, targetUserID uint, reason string) error
	RemovePost(moderatorID, postID uint, reason string) error
	RemoveComment(moderatorID, commentID uint, reason string) error
	GetLogs(limit int) ([]*model.ModerationLog, error)
	GetLogsByCommunity(communityID uint, limit int) ([]*model.ModerationLog, error)
	CreateModerationLog(modID uint, modUsername, action, targetType string, targetID uint, reason string) error

	CreateReport(reporterID uint, targetType string, targetID uint, reason string, description string) error
	GetReports() ([]*model.Report, error)
	GetReportByID(id uint) (*model.Report, error)
	ResolveReport(moderatorID uint, reportID uint, moderatorResponse string) error
	RejectReport(moderatorID uint, reportID uint, moderatorResponse string) error

	// Keyword filters
	AddKeywordFilter(modID uint, pattern string, isRegex bool, action string) error
	RemoveKeywordFilter(modID, filterID uint) error
	ListKeywordFilters() ([]*model.KeywordFilter, error)
	// CheckContent scans text against all active keyword filters.
	// Returns (matched bool, action string, matchedPattern string).
	// action is "block" or "shadow". "shadow" means hide content; it does NOT shadow-ban the author.
	CheckContent(text string) (bool, string, string)

	// Anti-spam
	RecordAction(userID uint, actionType string) (isSuspicious bool)
}

type moderationService struct {
	modRepo     repository.ModerationRepository
	userRepo    repository.UserRepository
	postRepo    repository.PostRepository
	commentRepo repository.CommentRepository
	commRepo    repository.CommunityRepository
	notifRepo   repository.NotificationRepository
	filterRepo  repository.KeywordFilterRepository

	// compiled keyword filters cache
	filterMu      sync.RWMutex
	compiledRules []compiledRule

	// anti-spam sliding window: userID -> actionType -> []timestamp
	spamMu      sync.Mutex
	spamWindows map[uint]map[string][]time.Time
}

type compiledRule struct {
	filter  *model.KeywordFilter
	regexp  *regexp.Regexp // non-nil only when IsRegex=true
	plain   string         // lower-cased plain text when IsRegex=false
}

// spamThresholds maps action type -> (window duration, max allowed count).
var spamThresholds = map[string]struct {
	window time.Duration
	max    int
}{
	"message": {30 * time.Second, 10},
	"post":    {5 * time.Minute, 3},
	"comment": {1 * time.Minute, 10},
}

func NewModerationService(
	modRepo repository.ModerationRepository,
	userRepo repository.UserRepository,
	postRepo repository.PostRepository,
	commentRepo repository.CommentRepository,
	commRepo repository.CommunityRepository,
	notifRepo repository.NotificationRepository,
	filterRepo repository.KeywordFilterRepository,
) ModerationService {
	svc := &moderationService{
		modRepo:     modRepo,
		userRepo:    userRepo,
		postRepo:    postRepo,
		commentRepo: commentRepo,
		commRepo:    commRepo,
		notifRepo:   notifRepo,
		filterRepo:  filterRepo,
		spamWindows: make(map[uint]map[string][]time.Time),
	}
	svc.reloadFilters()
	return svc
}

func (s *moderationService) requireAdminOrMod(moderatorID uint) (*model.User, error) {
	mod, err := s.userRepo.GetByID(moderatorID)
	if err != nil {
		return nil, errors.New("moderator not found")
	}
	if mod.Role != "admin" && mod.Role != "moderator" {
		return nil, errors.New("insufficient permissions: admin or moderator role required")
	}
	return mod, nil
}

func (s *moderationService) BanUser(moderatorID, targetUserID uint, reason string) error {
	mod, err := s.requireAdminOrMod(moderatorID)
	if err != nil {
		return err
	}

	target, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		return errors.New("target user not found")
	}
	if target.IsBanned {
		return errors.New("user is already banned")
	}

	target.IsBanned = true
	if err := s.userRepo.Update(target); err != nil {
		return err
	}

	return s.modRepo.CreateLog(&model.ModerationLog{
		ActorID:           moderatorID,
		TargetID:          targetUserID,
		TargetType:        "user",
		Action:            "user_banned",
		Details:           reason,
		ModeratorUsername: mod.Username,
	})
}

func (s *moderationService) UnbanUser(moderatorID, targetUserID uint, reason string) error {
	mod, err := s.requireAdminOrMod(moderatorID)
	if err != nil {
		return err
	}

	target, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		return errors.New("target user not found")
	}

	target.IsBanned = false
	if err := s.userRepo.Update(target); err != nil {
		return err
	}

	return s.modRepo.CreateLog(&model.ModerationLog{
		ActorID:           moderatorID,
		TargetID:          targetUserID,
		TargetType:        "user",
		Action:            "user_unbanned",
		Details:           reason,
		ModeratorUsername: mod.Username,
	})
}

func (s *moderationService) ShadowBanUser(moderatorID, targetUserID uint, reason string) error {
	mod, err := s.requireAdminOrMod(moderatorID)
	if err != nil {
		return err
	}

	target, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		return errors.New("target user not found")
	}
	if target.IsShadowBanned {
		return errors.New("user is already shadow-banned")
	}

	target.IsShadowBanned = true
	if err := s.userRepo.Update(target); err != nil {
		return err
	}

	return s.modRepo.CreateLog(&model.ModerationLog{
		ActorID:           moderatorID,
		TargetID:          targetUserID,
		TargetType:        "user",
		Action:            "user_shadow_banned",
		Details:           reason,
		ModeratorUsername: mod.Username,
	})
}

func (s *moderationService) UnshadowBanUser(moderatorID, targetUserID uint, reason string) error {
	mod, err := s.requireAdminOrMod(moderatorID)
	if err != nil {
		return err
	}

	target, err := s.userRepo.GetByID(targetUserID)
	if err != nil {
		return errors.New("target user not found")
	}

	target.IsShadowBanned = false
	if err := s.userRepo.Update(target); err != nil {
		return err
	}

	return s.modRepo.CreateLog(&model.ModerationLog{
		ActorID:           moderatorID,
		TargetID:          targetUserID,
		TargetType:        "user",
		Action:            "user_shadow_unbanned",
		Details:           reason,
		ModeratorUsername: mod.Username,
	})
}

func (s *moderationService) RemovePost(moderatorID, postID uint, reason string) error {
	post, err := s.postRepo.GetByID(postID)
	if err != nil {
		return errors.New("post not found")
	}

	isAuthorized := false
	user, err := s.userRepo.GetByID(moderatorID)
	if err == nil && (user.Role == "admin" || user.Role == "moderator") {
		isAuthorized = true
	} else {
		comm, err := s.commRepo.GetByID(post.CommunityID)
		if err == nil && comm.OwnerID == moderatorID {
			isAuthorized = true
		}
	}

	if !isAuthorized {
		return errors.New("insufficient permissions: admin, moderator or community owner role required")
	}

	if post.Status == "removed" {
		return errors.New("post is already removed")
	}

	post.Status = "removed"
	if err := s.postRepo.Update(post); err != nil {
		return err
	}

	var modUsername string
	if user != nil {
		modUsername = user.Username
	}

	return s.modRepo.CreateLog(&model.ModerationLog{
		ActorID:           moderatorID,
		TargetID:          postID,
		TargetType:        "post",
		Action:            "content_removed",
		Details:           reason,
		ModeratorUsername: modUsername,
	})
}

func (s *moderationService) RemoveComment(moderatorID, commentID uint, reason string) error {
	comment, err := s.commentRepo.GetByID(commentID)
	if err != nil {
		return errors.New("comment not found")
	}

	post, err := s.postRepo.GetByID(comment.PostID)
	if err != nil {
		return errors.New("associated post not found")
	}

	isAuthorized := false
	user, err := s.userRepo.GetByID(moderatorID)
	if err == nil && (user.Role == "admin" || user.Role == "moderator") {
		isAuthorized = true
	} else {
		comm, err := s.commRepo.GetByID(post.CommunityID)
		if err == nil && comm.OwnerID == moderatorID {
			isAuthorized = true
		}
	}

	if !isAuthorized {
		return errors.New("insufficient permissions: admin, moderator or community owner role required")
	}

	comment.IsDeleted = true
	comment.Content = "[удалено модератором]"
	if err := s.commentRepo.Update(comment); err != nil {
		return err
	}

	var modUsername string
	if user != nil {
		modUsername = user.Username
	}

	return s.modRepo.CreateLog(&model.ModerationLog{
		ActorID:           moderatorID,
		TargetID:          commentID,
		TargetType:        "comment",
		Action:            "content_removed",
		Details:           reason,
		ModeratorUsername: modUsername,
	})
}

func (s *moderationService) GetLogs(limit int) ([]*model.ModerationLog, error) {
	return s.modRepo.GetLogs(limit)
}

func (s *moderationService) GetLogsByCommunity(communityID uint, limit int) ([]*model.ModerationLog, error) {
	return s.modRepo.GetLogsByCommunity(communityID, limit)
}

func (s *moderationService) CreateModerationLog(modID uint, modUsername, action, targetType string, targetID uint, reason string) error {
	return s.modRepo.CreateLog(&model.ModerationLog{
		ActorID:           modID,
		ModeratorUsername: modUsername,
		Action:            action,
		TargetType:        targetType,
		TargetID:          targetID,
		Details:           reason,
	})
}

// ============================================================
// Keyword Filters
// ============================================================

// reloadFilters compiles all keyword filter patterns from DB into memory.
func (s *moderationService) reloadFilters() {
	filters, err := s.filterRepo.List()
	if err != nil {
		return
	}
	rules := make([]compiledRule, 0, len(filters))
	for _, f := range filters {
		r := compiledRule{filter: f}
		if f.IsRegex {
			if re, err := regexp.Compile("(?i)" + f.Pattern); err == nil {
				r.regexp = re
			}
		} else {
			r.plain = strings.ToLower(f.Pattern)
		}
		rules = append(rules, r)
	}
	s.filterMu.Lock()
	s.compiledRules = rules
	s.filterMu.Unlock()
}

func (s *moderationService) AddKeywordFilter(modID uint, pattern string, isRegex bool, action string) error {
	if action != "block" && action != "shadow" {
		return errors.New("action must be 'block' or 'shadow'")
	}
	if isRegex {
		if _, err := regexp.Compile(pattern); err != nil {
			return errors.New("invalid regex pattern: " + err.Error())
		}
	}
	f := &model.KeywordFilter{
		Pattern:   pattern,
		IsRegex:   isRegex,
		Action:    action,
		CreatedBy: modID,
	}
	if err := s.filterRepo.Create(f); err != nil {
		return err
	}
	s.reloadFilters()
	return nil
}

func (s *moderationService) RemoveKeywordFilter(modID, filterID uint) error {
	if err := s.filterRepo.Delete(filterID); err != nil {
		return err
	}
	s.reloadFilters()
	return nil
}

func (s *moderationService) ListKeywordFilters() ([]*model.KeywordFilter, error) {
	return s.filterRepo.List()
}

// CheckContent checks text against all compiled keyword filter rules.
// Returns (matched, action, matchedPattern).
// action is "block" (reject content) or "shadow" (hide content from others — does NOT affect the author's account).
func (s *moderationService) CheckContent(text string) (bool, string, string) {
	s.filterMu.RLock()
	rules := s.compiledRules
	s.filterMu.RUnlock()

	lowerText := strings.ToLower(text)
	for _, r := range rules {
		if r.regexp != nil {
			if r.regexp.MatchString(text) {
				return true, r.filter.Action, r.filter.Pattern
			}
		} else if r.plain != "" {
			if strings.Contains(lowerText, r.plain) {
				return true, r.filter.Action, r.filter.Pattern
			}
		}
	}
	return false, "", ""
}

// ============================================================
// Anti-Spam (sliding window rate limiter)
// ============================================================

// RecordAction records a user action and returns true if the user has exceeded
// spam thresholds and should be flagged as suspicious (triggering Turnstile CAPTCHA).
// Suspicious users are NOT shadow-banned — their content is still visible.
func (s *moderationService) RecordAction(userID uint, actionType string) bool {
	threshold, ok := spamThresholds[actionType]
	if !ok {
		return false
	}

	now := time.Now()
	windowStart := now.Add(-threshold.window)

	s.spamMu.Lock()
	if s.spamWindows[userID] == nil {
		s.spamWindows[userID] = make(map[string][]time.Time)
	}
	// Append current action and prune expired entries
	timestamps := append(s.spamWindows[userID][actionType], now)
	clean := timestamps[:0]
	for _, t := range timestamps {
		if t.After(windowStart) {
			clean = append(clean, t)
		}
	}
	s.spamWindows[userID][actionType] = clean
	count := len(clean)
	s.spamMu.Unlock()

	if count > threshold.max {
		// Flag user as suspicious asynchronously
		go func() {
			if user, err := s.userRepo.GetByID(userID); err == nil && !user.IsSuspicious {
				user.IsSuspicious = true
				_ = s.userRepo.Update(user)
				_ = s.modRepo.CreateLog(&model.ModerationLog{
					ActorID:    0,
					TargetID:   userID,
					TargetType: "user",
					Action:     "auto_spam_detected",
					Details:    "Exceeded " + actionType + " rate limit",
				})
			}
		}()
		return true
	}
	return false
}

func (s *moderationService) CreateReport(reporterID uint, targetType string, targetID uint, reason string, description string) error {
	reporter, _ := s.userRepo.GetByID(reporterID)
	username := "user"
	if reporter != nil {
		username = reporter.Username
	}

	report := &model.Report{
		ReporterID:       reporterID,
		ReporterUsername: username,
		TargetID:         targetID,
		TargetType:       targetType,
		Reason:           reason,
		Description:      description,
		Status:           "pending",
	}

	return s.modRepo.CreateReport(report)
}

func (s *moderationService) GetReports() ([]*model.Report, error) {
	return s.modRepo.GetReports()
}

func (s *moderationService) GetReportByID(id uint) (*model.Report, error) {
	return s.modRepo.GetReportByID(id)
}

func (s *moderationService) ResolveReport(moderatorID uint, reportID uint, moderatorResponse string) error {
	mod, err := s.requireAdminOrMod(moderatorID)
	if err != nil {
		return err
	}

	report, err := s.modRepo.GetReportByID(reportID)
	if err != nil {
		return err
	}

	report.Status = "resolved"
	report.ModeratorResponse = moderatorResponse
	err = s.modRepo.UpdateReport(report)
	if err != nil {
		return err
	}

	// Perform action based on target type
	if report.TargetType == "post" {
		post, err := s.postRepo.GetByID(report.TargetID)
		if err == nil && post.Status != "removed" {
			post.Status = "removed"
			_ = s.postRepo.Update(post)
			// Notify author
			notif := &model.Notification{
				UserID:      post.AuthorID,
				Type:        "content_removed",
				Title:       "Публикация удалена",
				Body:        "Ваша публикация '" + post.Title + "' была удалена модератором: " + moderatorResponse,
			}
			_ = s.notifRepo.Create(notif)
		}
	} else if report.TargetType == "comment" {
		comment, err := s.commentRepo.GetByID(report.TargetID)
		if err == nil && !comment.IsDeleted {
			comment.IsDeleted = true
			comment.Content = "[удалено модератором]"
			_ = s.commentRepo.Update(comment)
			// Notify author
			notif := &model.Notification{
				UserID:      comment.AuthorID,
				Type:        "content_removed",
				Title:       "Комментарий удален",
				Body:        "Ваш комментарий был удален модератором: " + moderatorResponse,
			}
			_ = s.notifRepo.Create(notif)
		}
	} else if report.TargetType == "user" {
		targetUser, err := s.userRepo.GetByID(report.TargetID)
		if err == nil && !targetUser.IsBanned {
			targetUser.IsBanned = true
			_ = s.userRepo.Update(targetUser)
			// Notify user
			notif := &model.Notification{
				UserID:      targetUser.ID,
				Type:        "user_banned",
				Title:       "Аккаунт заблокирован",
				Body:        "Ваш аккаунт был заблокирован модератором: " + moderatorResponse,
			}
			_ = s.notifRepo.Create(notif)
		}
	}

	// Notify reporter
	notif := &model.Notification{
		UserID:      report.ReporterID,
		Type:        "report_resolved",
		Title:       "Жалоба рассмотрена",
		Body:        "Ваша жалоба на тему '" + report.Reason + "' была одобрена модератором: " + moderatorResponse,
	}
	_ = s.notifRepo.Create(notif)

	// Write audit log
	_ = s.modRepo.CreateLog(&model.ModerationLog{
		ActorID:           moderatorID,
		TargetID:          reportID,
		TargetType:        "report",
		Action:            "report_resolved",
		Details:           "Report resolved: " + moderatorResponse,
		ModeratorUsername: mod.Username,
	})

	return nil
}

func (s *moderationService) RejectReport(moderatorID uint, reportID uint, moderatorResponse string) error {
	mod, err := s.requireAdminOrMod(moderatorID)
	if err != nil {
		return err
	}

	report, err := s.modRepo.GetReportByID(reportID)
	if err != nil {
		return err
	}

	report.Status = "rejected"
	report.ModeratorResponse = moderatorResponse
	err = s.modRepo.UpdateReport(report)
	if err != nil {
		return err
	}

	// Notify reporter
	notif := &model.Notification{
		UserID:      report.ReporterID,
		Type:        "report_rejected",
		Title:       "Жалоба отклонена",
		Body:        "Ваша жалоба на тему '" + report.Reason + "' была отклонена модератором: " + moderatorResponse,
	}
	_ = s.notifRepo.Create(notif)

	// Write audit log
	_ = s.modRepo.CreateLog(&model.ModerationLog{
		ActorID:           moderatorID,
		TargetID:          reportID,
		TargetType:        "report",
		Action:            "report_rejected",
		Details:           "Report rejected: " + moderatorResponse,
		ModeratorUsername: mod.Username,
	})

	return nil
}
