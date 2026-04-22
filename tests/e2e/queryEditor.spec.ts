import { test, expect, type ExplorePage } from '@grafana/plugin-e2e';

const DS_UID = 'jaeger';
const DS_TYPE = 'grafana-jaeger-datasource';

function exploreUrl(uid: string, query: Record<string, unknown> = {}): string {
  const panes = JSON.stringify({
    a: {
      datasource: uid,
      queries: [{ refId: 'A', datasource: { type: DS_TYPE, uid }, ...query }],
      range: { from: 'now-1h', to: 'now' },
    },
  });
  return `/explore?orgId=1&schemaVersion=1&panes=${encodeURIComponent(panes)}`;
}

// TODO: remove once @grafana/plugin-e2e exposes body reading natively
async function waitForQueryDataResponseWithBody(explorePage: ExplorePage) {
  let body: Record<string, unknown> | null = null;
  const responsePromise = explorePage.waitForQueryDataResponse(async (r) => {
    if (!r.ok()) return false;
    const b: any = await r.json().catch(() => null);
    if (!Array.isArray(b?.results?.A?.frames)) return false;
    body = b;
    return true;
  });
  return { responsePromise, getBody: () => body };
}

test.describe('Query editor', () => {
  test.describe('rendering', () => {
    test(
      'smoke: renders query type options',
      { tag: '@plugins' },
      async ({ page, readProvisionedDataSource }) => {
        const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
        await page.goto(exploreUrl(ds.uid ?? DS_UID));
        await expect(page.getByRole('radio', { name: 'Search' })).toBeVisible();
        await expect(page.getByRole('radio', { name: 'TraceID' })).toBeVisible();
        await expect(page.getByRole('radio', { name: 'Dependency graph' })).toBeVisible();
      }
    );

    test('renders Trace ID label in default mode', async ({ page, readProvisionedDataSource }) => {
      const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
      await page.goto(exploreUrl(ds.uid ?? DS_UID));
      await expect(page.getByRole('radio', { name: 'TraceID' })).toBeChecked();
      await expect(page.getByText('Trace ID', { exact: true })).toBeVisible();
    });
  });

  test.describe('Search mode', () => {
    test('shows all Search form fields', async ({ page, readProvisionedDataSource }) => {
      const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
      await page.goto(exploreUrl(ds.uid ?? DS_UID, { queryType: 'search' }));
      await expect(page.getByRole('radio', { name: 'Search' })).toBeChecked();
      await expect(page.getByRole('combobox', { name: 'select-service-name' })).toBeVisible();
      await expect(page.getByRole('textbox', { name: 'Tags' })).toBeVisible();
      await expect(page.getByRole('textbox', { name: 'Min Duration' })).toBeVisible();
      await expect(page.getByRole('textbox', { name: 'Max Duration' })).toBeVisible();
      await expect(page.getByRole('spinbutton', { name: 'Limit' })).toBeVisible();
    });
  });

  test.describe('Dependency graph mode', () => {
    test('shows no form input fields', async ({ page, readProvisionedDataSource }) => {
      const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
      await page.goto(exploreUrl(ds.uid ?? DS_UID, { queryType: 'dependencyGraph' }));
      await expect(page.getByRole('radio', { name: 'Dependency graph' })).toBeChecked();
      await expect(page.getByRole('combobox', { name: 'select-service-name' })).not.toBeVisible();
      await expect(page.getByRole('textbox', { name: 'Tags' })).not.toBeVisible();
    });
  });
});

test.describe('Query editor with live backend', () => {
  test.describe.configure({ mode: 'serial' });

  test('Search: returns traces for jaeger-all-in-one service', async ({
    page,
    explorePage,
    readProvisionedDataSource,
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    // Query fires on load — register before goto
    const { responsePromise, getBody } = await waitForQueryDataResponseWithBody(explorePage);
    await page.goto(exploreUrl(ds.uid ?? DS_UID, { queryType: 'search', service: 'jaeger-all-in-one' }));
    await responsePromise;
    expect((getBody() as any)?.results?.A?.frames?.length).toBeGreaterThan(0);
  });

  test('Search: returns traces when filtering by operation', async ({
    page,
    explorePage,
    readProvisionedDataSource,
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    const { responsePromise, getBody } = await waitForQueryDataResponseWithBody(explorePage);
    await page.goto(
      exploreUrl(ds.uid ?? DS_UID, {
        queryType: 'search',
        service: 'jaeger-all-in-one',
        operation: '/api/traces',
      })
    );
    await responsePromise;
    const frames: any[] = (getBody() as any)?.results?.A?.frames ?? [];
    expect(frames.length).toBeGreaterThan(0);
    // All returned trace names should contain the operation name
    const traceNames: string[] = frames[0]?.data?.values?.[1] ?? [];
    expect(traceNames.every((n: string) => n.includes('/api/traces'))).toBe(true);
  });

  test('Dependency graph: returns node graph frames', async ({
    page,
    explorePage,
    readProvisionedDataSource,
  }) => {
    const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
    const { responsePromise, getBody } = await waitForQueryDataResponseWithBody(explorePage);
    await page.goto(exploreUrl(ds.uid ?? DS_UID, { queryType: 'dependencyGraph' }));
    await responsePromise;
    expect((getBody() as any)?.results?.A?.frames?.length).toBeGreaterThan(0);
  });
});
