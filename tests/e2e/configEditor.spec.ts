import { test, expect } from '@grafana/plugin-e2e';

test.describe('Config editor', () => {
  test.describe('rendering', () => {
    test(
      'smoke: should render config editor',
      { tag: '@plugins' },
      async ({ createDataSourceConfigPage, page }) => {
        await createDataSourceConfigPage({ type: 'jaeger' });
        await expect(page.getByText('Type: Jaeger', { exact: true })).toBeVisible();
        await expect(page.getByRole('heading', { name: 'Connection', exact: true })).toBeVisible();
      }
    );

    test('should render Connection section', async ({ createDataSourceConfigPage, page }) => {
      await createDataSourceConfigPage({ type: 'jaeger' });
      await expect(page.getByRole('heading', { name: 'Connection', exact: true })).toBeVisible();
      await expect(page.getByRole('textbox', { name: 'Data source connection URL' })).toBeVisible();
    });

    test('should render Authentication section', async ({ createDataSourceConfigPage, page }) => {
      await createDataSourceConfigPage({ type: 'jaeger' });
      await expect(page.getByRole('heading', { name: 'Authentication', exact: true })).toBeVisible();
      await expect(page.getByRole('combobox', { name: 'Authentication method' })).toBeVisible();
    });

    test('should render Trace to logs section', async ({ createDataSourceConfigPage, page }) => {
      await createDataSourceConfigPage({ type: 'jaeger' });
      await expect(page.getByRole('heading', { name: 'Trace to logs', exact: true })).toBeVisible();
    });

    test('should render Trace to metrics section', async ({ createDataSourceConfigPage, page }) => {
      await createDataSourceConfigPage({ type: 'jaeger' });
      await expect(page.getByRole('heading', { name: 'Trace to metrics', exact: true })).toBeVisible();
    });
  });

  test.describe('provisioned datasource', () => {
    test('should load provisioned datasource config', async ({ readProvisionedDataSource, gotoDataSourceConfigPage, page }) => {
      const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
      await gotoDataSourceConfigPage(ds.uid ?? 'jaeger');
      await expect(page.getByText('Type: Jaeger', { exact: true })).toBeVisible();
      // URL set in provisioning/datasources/datasources.yml; in cloud CI overridden via DS_INSTANCE_HOST
      const expectedUrl = process.env.DS_INSTANCE_HOST
        ? `http://${process.env.DS_INSTANCE_HOST}:${process.env.DS_INSTANCE_PORT ?? '16686'}`
        : 'http://jaeger:16686';
      await expect(page.getByRole('textbox', { name: 'Data source connection URL' })).toHaveValue(expectedUrl);
    });
  });

  test.describe('save & test', () => {
    test('should pass health check for provisioned datasource', async ({ readProvisionedDataSource, gotoDataSourceConfigPage }) => {
      const ds = await readProvisionedDataSource({ fileName: 'datasources.yml' });
      const configPage = await gotoDataSourceConfigPage(ds.uid ?? 'jaeger');
      await expect(configPage.saveAndTest()).toBeOK();
    });

    test('should show error when backend is unreachable', async ({ createDataSourceConfigPage, page }) => {
      const configPage = await createDataSourceConfigPage({ type: 'jaeger' });
      await page.getByRole('textbox', { name: 'Data source connection URL' }).fill('http://localhost:1');
      await expect(configPage.saveAndTest()).not.toBeOK();
      await expect(configPage).toHaveAlert('error');
    });
  });
});
