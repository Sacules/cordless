package ui

import (
	"sort"

	"github.com/Bios-Marcel/cordless/internal/config"
	"github.com/Bios-Marcel/cordless/internal/discordgoplus"
	"github.com/Bios-Marcel/discordgo"
	"github.com/Bios-Marcel/tview"
	"github.com/gdamore/tcell"
)

type UserTree struct {
	internalTreeView *tview.TreeView
	rootNode         *tview.TreeNode

	state *discordgo.State

	userNodes map[string]*tview.TreeNode

	roleNodes map[string]*tview.TreeNode
	roles     []*discordgo.Role
}

func NewUserTree(state *discordgo.State) *UserTree {
	userTree := &UserTree{
		state:            state,
		userNodes:        make(map[string]*tview.TreeNode),
		roleNodes:        make(map[string]*tview.TreeNode),
		roles:            make([]*discordgo.Role, 0),
		rootNode:         tview.NewTreeNode(""),
		internalTreeView: tview.NewTreeView(),
	}

	userTree.internalTreeView.
		SetVimBindingsEnabled(config.GetConfig().OnTypeInListBehaviour == config.DoNothingOnTypeInList).
		SetRoot(userTree.rootNode).
		SetTopLevel(1).
		SetCycleSelection(true)
	userTree.internalTreeView.SetBorder(true)

	return userTree
}

func (userTree *UserTree) LoadGuild(guildID string) error {
	userTree.userNodes = make(map[string]*tview.TreeNode)
	userTree.roleNodes = make(map[string]*tview.TreeNode)
	userTree.roles = make([]*discordgo.Role, 0)

	userTree.rootNode.ClearChildren()

	guildRoles, roleLoadError := userTree.loadGuildRoles(guildID)
	if roleLoadError != nil {
		return roleLoadError
	}
	userTree.roles = guildRoles

	userLoadError := userTree.loadGuildMembers(guildID)
	if userLoadError != nil {
		return userLoadError
	}

	userTree.selectFirstNode()

	return nil
}

func (userTree *UserTree) selectFirstNode() {
	if userTree.internalTreeView.GetCurrentNode() == nil {
		userNodes := userTree.rootNode.GetChildren()
		if userNodes != nil && len(userNodes) > 0 {
			userTree.internalTreeView.SetCurrentNode(userTree.rootNode.GetChildren()[0])
		}
	}
}

func (userTree *UserTree) loadGuildMembers(guildID string) error {
	members, stateError := userTree.state.Members(guildID)
	if stateError != nil {
		return stateError
	}

	userTree.AddOrUpdateMembers(members)

	return nil
}

func (userTree *UserTree) loadGuildRoles(guildID string) ([]*discordgo.Role, error) {
	guild, stateError := userTree.state.Guild(guildID)
	if stateError != nil {
		return nil, stateError
	}

	guildRoles := guild.Roles

	sort.Slice(guildRoles, func(a, b int) bool {
		return guildRoles[a].Position > guildRoles[b].Position
	})

	for _, role := range guildRoles {
		if role.Hoist {
			roleNode := tview.NewTreeNode(role.Name)
			roleNode.SetSelectable(false)
			userTree.roleNodes[role.ID] = roleNode
			userTree.rootNode.AddChild(roleNode)
		}
	}

	return guildRoles, nil
}

func (userTree *UserTree) AddOrUpdateMember(member *discordgo.Member) {
	nameToUse := discordgoplus.GetMemberName(member, nil)

	userNode, contains := userTree.userNodes[member.User.ID]
	if contains && userNode != nil {
		userNode.SetText(nameToUse)
		return
	}

	userNode = tview.NewTreeNode(nameToUse)
	userTree.userNodes[member.User.ID] = userNode

	discordgoplus.SortUserRoles(member.Roles, userTree.roles)

	for _, userRole := range member.Roles {
		roleNode, exists := userTree.roleNodes[userRole]
		if exists && roleNode != nil {
			roleNode.AddChild(userNode)
			return
		}
	}

	userTree.rootNode.AddChild(userNode)
}

func (userTree *UserTree) AddOrUpdateMembers(members []*discordgo.Member) {
	for _, member := range members {
		userTree.AddOrUpdateMember(member)
	}
}

func (userTree *UserTree) RemoveMember(member *discordgo.Member) {
	userNode, contains := userTree.userNodes[member.User.ID]
	if contains {
		userTree.rootNode.Walk(func(node, parent *tview.TreeNode) bool {
			if node == userNode {
				if len(parent.GetChildren()) == 1 {
					parent.SetChildren(make([]*tview.TreeNode, 0))
				} else {
					indexToDelete := -1
					for index, child := range parent.GetChildren() {
						if child == node {
							indexToDelete = index
							break
						}
					}

					if indexToDelete == 0 {
						parent.SetChildren(parent.GetChildren()[1:])
					} else if indexToDelete == len(parent.GetChildren())-1 {
						parent.SetChildren(parent.GetChildren()[:len(parent.GetChildren())-1])
					} else {
						parent.SetChildren(append(parent.GetChildren()[0:indexToDelete],
							parent.GetChildren()[indexToDelete+1:]...))
					}
				}

				return false
			}

			return true
		})
	}
}

func (userTree *UserTree) RemoveMembers(members []*discordgo.Member) {
	for _, member := range members {
		userTree.RemoveMember(member)
	}
}

//SetInputCapture delegates to tviews SetInputCapture
func (userTree *UserTree) SetInputCapture(capture func(event *tcell.EventKey) *tcell.EventKey) {
	userTree.internalTreeView.SetInputCapture(capture)
}
