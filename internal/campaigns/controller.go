package campaigns

import (
	"net/http"
	"strconv"
	"strings"

	"dndmanager/internal/platform"
)

// CharacterSummary is the campaign page's character row (avoids importing characters).
type CharacterSummary struct {
	ID      int64
	Name    string
	Level   int
	Owner   string
	OwnerID int64
}

// CharacterLister is implemented by characters.Service.
type CharacterLister interface {
	ListByCampaign(campaignID int64) ([]CharacterSummary, error)
}

// BattleFinder is implemented by battles.Service (avoids an import cycle).
type BattleFinder interface {
	HasBattle(campaignID int64) (bool, error)
}

// Controller serves campaign list, create, join, and the table page.
type Controller struct {
	Svc        *Service
	Characters CharacterLister
	Battles    BattleFinder
	Render     *platform.Renderer
}

// Mount registers authenticated campaign routes.
func (c *Controller) Mount(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /campaigns", auth(http.HandlerFunc(c.list)))
	mux.Handle("GET /campaigns/new", auth(http.HandlerFunc(c.showCreate)))
	mux.Handle("POST /campaigns", auth(http.HandlerFunc(c.create)))
	mux.Handle("GET /campaigns/join", auth(http.HandlerFunc(c.showJoin)))
	mux.Handle("POST /campaigns/join", auth(http.HandlerFunc(c.join)))
	mux.Handle("GET /campaigns/{id}", auth(http.HandlerFunc(c.show)))
}

func (c *Controller) base(r *http.Request, title string) platform.BaseView {
	return platform.NewBase(r, title)
}

type listView struct {
	platform.BaseView
	Campaigns []CampaignListItem
}

func (c *Controller) list(w http.ResponseWriter, r *http.Request) {
	u := platform.UserFrom(r.Context())
	items, err := c.Svc.ListForUser(u.ID)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	v := listView{BaseView: c.base(r, "nav.campaigns"), Campaigns: items}
	c.Render.Render(w, "campaigns/list.html", v.Lang, http.StatusOK, v)
}

type formView struct {
	platform.BaseView
	Name   string
	Invite string
}

func (c *Controller) showCreate(w http.ResponseWriter, r *http.Request) {
	v := formView{BaseView: c.base(r, "campaign.create.title")}
	c.Render.Render(w, "campaigns/new.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) create(w http.ResponseWriter, r *http.Request) {
	u := platform.UserFrom(r.Context())
	name := platform.FormTrim(r, "name")
	camp, err := c.Svc.Create(name, u.ID)
	if err != nil {
		v := formView{BaseView: c.base(r, "campaign.create.title"), Name: name}
		v.Error = ErrorKey(err)
		c.Render.Render(w, "campaigns/new.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	platform.Redirect(w, r, "/campaigns/"+strconv.FormatInt(camp.ID, 10))
}

func (c *Controller) showJoin(w http.ResponseWriter, r *http.Request) {
	v := formView{BaseView: c.base(r, "campaign.join.title")}
	c.Render.Render(w, "campaigns/join.html", v.Lang, http.StatusOK, v)
}

func (c *Controller) join(w http.ResponseWriter, r *http.Request) {
	u := platform.UserFrom(r.Context())
	invite := platform.FormTrim(r, "invite")
	camp, err := c.Svc.Join(invite, u.ID)
	if err != nil {
		if err == ErrAlreadyMember {
			if existing, e2 := c.Svc.Repo.FindByInvite(strings.ToUpper(strings.TrimSpace(invite))); e2 == nil {
				platform.Redirect(w, r, "/campaigns/"+strconv.FormatInt(existing.ID, 10))
				return
			}
		}
		v := formView{BaseView: c.base(r, "campaign.join.title"), Invite: invite}
		v.Error = ErrorKey(err)
		c.Render.Render(w, "campaigns/join.html", v.Lang, http.StatusUnprocessableEntity, v)
		return
	}
	platform.Redirect(w, r, "/campaigns/"+strconv.FormatInt(camp.ID, 10))
}

type showView struct {
	platform.BaseView
	Campaign   *Campaign
	Members    []Membership
	Characters []CharacterSummary
	MemberRole string
	IsDM       bool
	HasBattle  bool
}

func (c *Controller) show(w http.ResponseWriter, r *http.Request) {
	id, err := platform.PathID(r, "id")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	u := platform.UserFrom(r.Context())
	camp, err := c.Svc.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	mem, err := c.Svc.RequireMember(id, u.ID)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	members, err := c.Svc.Members(id)
	if err != nil {
		http.Error(w, "error", http.StatusInternalServerError)
		return
	}
	var chars []CharacterSummary
	if c.Characters != nil {
		chars, err = c.Characters.ListByCampaign(id)
		if err != nil {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
	}
	hasBattle := false
	if c.Battles != nil {
		hasBattle, err = c.Battles.HasBattle(id)
		if err != nil {
			http.Error(w, "error", http.StatusInternalServerError)
			return
		}
	}
	v := showView{
		BaseView:   c.base(r, camp.Name),
		Campaign:   camp,
		Members:    members,
		Characters: chars,
		MemberRole: mem.Role,
		IsDM:       mem.Role == MemberDM,
		HasBattle:  hasBattle,
	}
	c.Render.Render(w, "campaigns/show.html", v.Lang, http.StatusOK, v)
}
