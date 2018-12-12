// @flow
import * as React from 'react'
import * as Types from '../constants/types/people'
import * as Kb from '../common-adapters'
import * as Styles from '../styles'
import Todo from './todo/container'
import FollowNotification from './follow-notification'
import Announcement from './announcement/container'
import FollowSuggestions from './follow-suggestions'
import {type Props} from '.'

export const itemToComponent: (Types.PeopleScreenItem, Props) => React.Node = (item, props) => {
  switch (item.type) {
    case 'todo':
      return (
        <Todo
          badged={item.badged}
          todoType={item.todoType}
          instructions={item.instructions}
          confirmLabel={item.confirmLabel}
          dismissable={item.dismissable}
          icon={item.icon}
          key={item.todoType}
        />
      )
    case 'notification':
      return (
        <FollowNotification
          type={item.type}
          newFollows={item.newFollows}
          notificationTime={item.notificationTime}
          badged={item.badged}
          numAdditional={item.numAdditional}
          key={item.notificationTime}
          onClickUser={props.onClickUser}
        />
      )
    case 'announcement':
      return (
        <Announcement
          appLink={item.appLink}
          badged={item.badged}
          confirmLabel={item.confirmLabel}
          dismissable={item.dismissable}
          iconUrl={item.iconUrl}
          key={item.text}
          text={item.text}
          url={item.url}
        />
      )
  }
  return null
}

export const PeoplePageSearchBar = (props: Props) => (
  <Kb.ClickableBox onClick={props.onSearch} style={styles.searchContainer}>
    <Kb.Icon
      color={Styles.globalColors.black_20}
      fontSize={20}
      style={styles.searchIcon}
      type="iconfont-search"
    />
    <Kb.Text style={styles.searchText} type="BodySemibold">
      Search people
    </Kb.Text>
  </Kb.ClickableBox>
)

export const PeoplePageList = (props: Props) => (
  <Kb.Box style={styles.list}>
    {props.newItems.map(item => itemToComponent(item, props))}
    <FollowSuggestions suggestions={props.followSuggestions} />
    {props.oldItems.map(item => itemToComponent(item, props))}
  </Kb.Box>
)

const styles = Styles.styleSheetCreate({
  list: Styles.platformStyles({
    common: {
      ...Styles.globalStyles.flexBoxColumn,
      position: 'relative',
      width: '100%',
    },
    isElectron: {
      marginTop: 48,
    },
  }),
  searchContainer: Styles.platformStyles({
    common: {
      ...Styles.globalStyles.flexBoxRow,
      alignItems: 'center',
      backgroundColor: Styles.globalColors.black_10,
      borderRadius: Styles.borderRadius,
      justifyContent: 'center',
      width: '100%',
    },
    isElectron: {
      ...Styles.desktopStyles.clickable,
      minHeight: 24,
      width: 240,
    },
    isMobile: {
      height: 32,
    },
  }),
  searchIcon: {
    position: 'relative',
    top: 1,
  },
  searchText: {
    color: Styles.globalColors.black_40,
    padding: Styles.globalMargins.xtiny,
  },
})
