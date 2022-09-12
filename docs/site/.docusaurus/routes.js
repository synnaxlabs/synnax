import React from 'react';
import ComponentCreator from '@docusaurus/ComponentCreator';

export default [
  {
    path: '/__docusaurus/debug',
    component: ComponentCreator('/__docusaurus/debug', '300'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/config',
    component: ComponentCreator('/__docusaurus/debug/config', '6e6'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/content',
    component: ComponentCreator('/__docusaurus/debug/content', '4b9'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/globalData',
    component: ComponentCreator('/__docusaurus/debug/globalData', '4e2'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/metadata',
    component: ComponentCreator('/__docusaurus/debug/metadata', 'adf'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/registry',
    component: ComponentCreator('/__docusaurus/debug/registry', '17c'),
    exact: true
  },
  {
    path: '/__docusaurus/debug/routes',
    component: ComponentCreator('/__docusaurus/debug/routes', 'c54'),
    exact: true
  },
  {
    path: '/',
    component: ComponentCreator('/', 'cd2'),
    routes: [
      {
        path: '/',
        component: ComponentCreator('/', '96c'),
        exact: true,
        sidebar: "tutorialSidebar"
      },
      {
        path: '/category/python-client',
        component: ComponentCreator('/category/python-client', 'b46'),
        exact: true,
        sidebar: "tutorialSidebar"
      },
      {
        path: '/category/technical-rfcs',
        component: ComponentCreator('/category/technical-rfcs', '849'),
        exact: true,
        sidebar: "tutorialSidebar"
      },
      {
        path: '/category/typescript-client',
        component: ComponentCreator('/category/typescript-client', '50c'),
        exact: true,
        sidebar: "tutorialSidebar"
      },
      {
        path: '/client-python/get-started',
        component: ComponentCreator('/client-python/get-started', 'e4c'),
        exact: true,
        sidebar: "tutorialSidebar"
      },
      {
        path: '/client-typescript/get-started',
        component: ComponentCreator('/client-typescript/get-started', '5d0'),
        exact: true,
        sidebar: "tutorialSidebar"
      },
      {
        path: '/concepts',
        component: ComponentCreator('/concepts', '1c6'),
        exact: true,
        sidebar: "tutorialSidebar"
      },
      {
        path: '/rfc/1-220517-cesium-segment-storage',
        component: ComponentCreator('/rfc/1-220517-cesium-segment-storage', 'fc6'),
        exact: true,
        sidebar: "tutorialSidebar"
      },
      {
        path: '/rfc/2-220518-aspen-distributed-storage',
        component: ComponentCreator('/rfc/2-220518-aspen-distributed-storage', '90f'),
        exact: true,
        sidebar: "tutorialSidebar"
      },
      {
        path: '/rfc/3-220604-segment-distribution',
        component: ComponentCreator('/rfc/3-220604-segment-distribution', '5a5'),
        exact: true,
        sidebar: "tutorialSidebar"
      },
      {
        path: '/rfc/4-220623-signal-gr',
        component: ComponentCreator('/rfc/4-220623-signal-gr', 'ff2'),
        exact: true,
        sidebar: "tutorialSidebar"
      },
      {
        path: '/rfc/5-220716-ontology',
        component: ComponentCreator('/rfc/5-220716-ontology', 'f05'),
        exact: true,
        sidebar: "tutorialSidebar"
      },
      {
        path: '/rfc/6-220809-freighter',
        component: ComponentCreator('/rfc/6-220809-freighter', '916'),
        exact: true,
        sidebar: "tutorialSidebar"
      },
      {
        path: '/rfc/7-220823-data-type',
        component: ComponentCreator('/rfc/7-220823-data-type', 'ede'),
        exact: true,
        sidebar: "tutorialSidebar"
      }
    ]
  },
  {
    path: '*',
    component: ComponentCreator('*'),
  },
];
