import { css } from '@emotion/css';
import React from 'react';
import { useAsync } from 'react-use';

import { GrafanaTheme2 } from '@grafana/data';
import { selectors } from '@grafana/e2e-selectors';
import { getBackendSrv } from '@grafana/runtime';
import { Alert, Spinner, useStyles2 } from '@grafana/ui';

import { contextSrv } from '../../../../core/services/context_srv';
import impressionSrv from '../../../../core/services/impression_srv';
import { GENERAL_FOLDER_UID } from '../../constants';
import { getGrafanaSearcher } from '../../service';

import { FolderSection } from './FolderSection';
import { SearchResultsProps } from './SearchResultsTable';

type Props = Pick<SearchResultsProps, 'selection' | 'selectionToggle' | 'onTagSelected' | 'onClickItem'> & {
  tags?: string[];
  hidePseudoFolders?: boolean;
};
export const RootFolderView = ({
  selection,
  selectionToggle,
  onTagSelected,
  tags,
  hidePseudoFolders,
  onClickItem,
}: Props) => {
  const styles = useStyles2(getStyles);

  const results = useAsync(async () => {
    const searcher = getGrafanaSearcher();
    const folders = await searcher.getFolderChildren();

    if (!hidePseudoFolders) {
      if (contextSrv.isSignedIn) {
        const stars = await getBackendSrv().get('api/user/stars');
        if (stars.length > 0) {
          folders.unshift({ title: 'Starred', icon: 'star', kind: 'folder', uid: '__starred', itemsUIDs: stars });
        }
      }

      const itemsUIDs = await impressionSrv.getDashboardOpened();
      if (itemsUIDs.length) {
        folders.unshift({ title: 'Recent', icon: 'clock-nine', kind: 'folder', uid: '__recent', itemsUIDs });
      }
    }

    folders.unshift({ title: 'General', url: '/dashboards', kind: 'folder', uid: GENERAL_FOLDER_UID });

    return folders;
  }, []);

  const renderResults = () => {
    if (results.loading) {
      return <Spinner className={styles.spinner} />;
    } else if (!results.value) {
      return <Alert className={styles.error} title={results.error ? results.error.message : 'Something went wrong'} />;
    } else {
      return results.value.map((section) => (
        <div data-testid={selectors.components.Search.sectionV2} className={styles.section} key={section.title}>
          {section.title && (
            <FolderSection
              selection={selection}
              selectionToggle={selectionToggle}
              onTagSelected={onTagSelected}
              section={section}
              tags={tags}
              onClickItem={onClickItem}
            />
          )}
        </div>
      ));
    }
  };

  return <div className={styles.wrapper}>{renderResults()}</div>;
};

const getStyles = (theme: GrafanaTheme2) => {
  return {
    wrapper: css`
      display: flex;
      flex-direction: column;
      overflow: auto;

      > ul {
        list-style: none;
      }

      border: solid 1px ${theme.v1.colors.border2};
    `,
    section: css`
      display: flex;
      flex-direction: column;
      background: ${theme.v1.colors.panelBg};

      &:not(:last-child) {
        border-bottom: solid 1px ${theme.v1.colors.border2};
      }
    `,
    spinner: css`
      align-items: center;
      display: flex;
      justify-content: center;
      min-height: 100px;
    `,
    error: css`
      margin: ${theme.spacing(4)} auto;
    `,
  };
};
