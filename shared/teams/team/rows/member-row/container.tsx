import * as Constants from '../../../../constants/teams'
import * as TeamsGen from '../../../../actions/teams-gen'
import * as Chat2Gen from '../../../../actions/chat2-gen'
import * as Tracker2Gen from '../../../../actions/tracker2-gen'
import * as ProfileGen from '../../../../actions/profile-gen'
import * as Types from '../../../../constants/types/teams'
import {TeamMemberRow} from '.'
import * as RouteTreeGen from '../../../../actions/route-tree-gen'
import {connect, isMobile} from '../../../../util/container'
import {anyWaiting} from '../../../../constants/waiting'

type OwnProps = {
  teamID: Types.TeamID
  username: string
}

const blankInfo = Constants.initialMemberInfo

export default connect(
  (state, {teamID, username}: OwnProps) => {
    const teamDetails = Constants.getTeamDetails(state, teamID)
    const {members: map = new Map(), teamname} = teamDetails
    const info = map.get(username) || blankInfo

    return {
      following: state.config.following.has(username),
      fullName: state.config.username === username ? 'You' : info.fullName,
      roleType: info.type,
      status: info.status,
      teamname,
      username: info.username,
      waitingForAdd: anyWaiting(state, Constants.addMemberWaitingKey(teamID, username)),
      waitingForRemove: anyWaiting(state, Constants.removeMemberWaitingKey(teamID, username)),
      you: state.config.username,
      youCanManageMembers: Constants.getCanPerform(state, teamname).manageMembers,
    }
  },
  (dispatch, {teamID, username}: OwnProps) => ({
    onChat: () =>
      username &&
      dispatch(Chat2Gen.createPreviewConversation({participants: [username], reason: 'teamMember'})),
    onClick: () =>
      dispatch(
        RouteTreeGen.createNavigateAppend({path: [{props: {teamID, username}, selected: 'teamMember'}]})
      ),
    onReAddToTeam: () =>
      dispatch(
        TeamsGen.createReAddToTeam({
          teamID,
          username,
        })
      ),
    onRemoveFromTeam: () => dispatch(TeamsGen.createRemoveMember({teamID, username})),
    onShowTracker: () =>
      dispatch(
        isMobile
          ? ProfileGen.createShowUserProfile({username})
          : Tracker2Gen.createShowUser({asTracker: true, username})
      ),
  }),
  (stateProps, dispatchProps, _: OwnProps) => ({
    following: stateProps.following,
    fullName: stateProps.fullName,
    onChat: dispatchProps.onChat,
    onClick: dispatchProps.onClick,
    onReAddToTeam: dispatchProps.onReAddToTeam,
    onRemoveFromTeam: dispatchProps.onRemoveFromTeam,
    onShowTracker: dispatchProps.onShowTracker,
    roleType: stateProps.roleType,
    status: stateProps.status,
    username: stateProps.username,
    waitingForAdd: stateProps.waitingForAdd,
    waitingForRemove: stateProps.waitingForRemove,
    you: stateProps.you,
    youCanManageMembers: stateProps.youCanManageMembers,
  })
)(TeamMemberRow)
