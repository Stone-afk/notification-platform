package domain

// AuditStatus 审核状态
type AuditStatus string

const (
	AuditStatusPending  AuditStatus = "PENDING"   // 待审核
	AuditStatusInReview AuditStatus = "IN_REVIEW" // 审核中
	AuditStatusRejected AuditStatus = "REJECTED"  // 已拒绝
	AuditStatusApproved AuditStatus = "APPROVED"  // 已通过
)

func (a AuditStatus) String() string {
	return string(a)
}

func (a AuditStatus) IsPending() bool {
	return a == AuditStatusPending
}

func (a AuditStatus) IsInReview() bool {
	return a == AuditStatusInReview
}

func (a AuditStatus) IsRejected() bool {
	return a == AuditStatusRejected
}

func (a AuditStatus) IsApproved() bool {
	return a == AuditStatusApproved
}

func (a AuditStatus) IsValid() bool {
	switch a {
	case AuditStatusPending, AuditStatusInReview, AuditStatusApproved, AuditStatusRejected:
		return true
	default:
		return false
	}
}
