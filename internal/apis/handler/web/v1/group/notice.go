package group

import (
	"errors"
	"fmt"
	"time"

	"go-chat/api/pb/web/v1"
	"go-chat/internal/entity"
	"go-chat/internal/pkg/core"
	"go-chat/internal/pkg/jsonutil"
	"go-chat/internal/repository/model"
	"go-chat/internal/repository/repo"
	"go-chat/internal/service"
	"go-chat/internal/service/message"
	"gorm.io/gorm"
)

type Notice struct {
	GroupMemberRepo    *repo.GroupMember
	GroupNoticeRepo    *repo.GroupNotice
	GroupMemberService service.IGroupMemberService
	Message            message.IService
	UsersRepo          *repo.Users
}

// CreateAndUpdate 添加或编辑群公告
func (c *Notice) CreateAndUpdate(ctx *core.Context) error {
	in := &web.GroupNoticeEditRequest{}
	if err := ctx.Context.ShouldBindJSON(in); err != nil {
		return ctx.InvalidParams(err)
	}

	uid := ctx.AuthId()

	if !c.GroupMemberRepo.IsMember(ctx.GetContext(), int(in.GroupId), uid, false) {
		return ctx.Error(entity.ErrPermissionDenied)
	}

	var (
		msg string
		err error
	)

	notice, err := c.GroupNoticeRepo.FindByWhere(ctx.GetContext(), "group_id = ?", in.GroupId)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return ctx.Error(err)
	}

	if notice == nil {
		msg = "群公告创建成功！"
		err = c.GroupNoticeRepo.Create(ctx.GetContext(), &model.GroupNotice{
			GroupId:      int(in.GroupId),
			CreatorId:    ctx.AuthId(),
			ModifyId:     ctx.AuthId(),
			Content:      in.Content,
			ConfirmUsers: "[]",
			IsConfirm:    2,
		})
	} else {
		msg = "群公告更新成功！"
		_, err = c.GroupNoticeRepo.UpdateByWhere(ctx.GetContext(), map[string]any{
			"content":    in.Content,
			"modify_id":  ctx.AuthId(),
			"updated_at": time.Now(),
		}, "group_id = ?", in.GroupId)
	}

	if err != nil {
		return ctx.Error(err)
	}

	userInfo, err := c.UsersRepo.FindByIdWithCache(ctx.GetContext(), uid)
	if err == nil {
		_ = c.Message.CreateGroupMessage(ctx.GetContext(), message.CreateGroupMessageOption{
			MsgType:  entity.ChatMsgTypeGroupNotice,
			FromId:   uid,
			ToFromId: int(in.GroupId),
			Extra: jsonutil.Encode(model.TalkRecordExtraGroupNotice{
				OwnerId:   uid,
				OwnerName: userInfo.Nickname,
				Title:     fmt.Sprintf("【%s】 更新了群公告", userInfo.Nickname),
				Content:   in.Content,
			}),
		})
	}

	return ctx.Success(nil, msg)
}
