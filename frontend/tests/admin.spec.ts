import { expect, test } from "@playwright/test";

const now = Date.now();
const hoursFromNow = (hours: number) =>
  new Date(now + hours * 60 * 60 * 1000).toISOString();

const requests = [
  {
    id: "101",
    userId: "1",
    houseId: "1",
    description: "Сильная протечка в первом подъезде",
    problemType: "PIPE_LEAK",
    responsibleOrganizationId: "1",
    responsibleOrganization: "УК «Тестовая»",
    status: "CREATED",
    deadline: hoursFromNow(6),
    createdAt: hoursFromNow(-1),
    address: "г. Москва, ул. Тестовая, д. 1",
    kind: "EMERGENCY",
    text: "В первом подъезде быстро течёт вода.",
    hasPhoto: false,
  },
  {
    id: "102",
    userId: "1",
    houseId: "1",
    description: "Не работает лифт",
    problemType: "ELEVATOR",
    responsibleOrganizationId: "2",
    responsibleOrganization: "Подрядчик «ТестЛифт»",
    status: "ACCEPTED",
    deadline: hoursFromNow(-28),
    createdAt: hoursFromNow(-72),
    address: "г. Москва, ул. Тестовая, д. 1, подъезд 2",
    kind: "PROBLEM",
    text: "Лифт не реагирует на кнопку вызова.",
    hasPhoto: false,
  },
  {
    id: "103",
    userId: "1",
    houseId: "1",
    description: "Когда будет уборка двора",
    problemType: "OTHER",
    responsibleOrganizationId: "1",
    responsibleOrganization: "УК «Тестовая»",
    status: "RESOLVED",
    deadline: hoursFromNow(-48),
    createdAt: hoursFromNow(-96),
    address: "г. Москва, ул. Тестовая, д. 1",
    kind: "QUESTION",
    text: "Закрытый вопрос про уборку.",
    hasPhoto: false,
  },
];

test.beforeEach(async ({ page }) => {
  await page.route("https://st.max.ru/js/max-web-app.js", (route) =>
    route.abort(),
  );
});

test("admin queue prioritizes work and keeps filters in the URL", async ({
  page,
}) => {
  await page.route("**/max/api/admin/requests", (route) =>
    route.fulfill({ contentType: "application/json", body: JSON.stringify(requests) }),
  );
  await page.route("**/max/api/organizations", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify([
        { id: "1", name: "УК «Тестовая»" },
        { id: "2", name: "Подрядчик «ТестЛифт»" },
      ]),
    }),
  );

  await page.goto("/max/admin");

  await expect(page.locator(".admin-request-card")).toHaveCount(2);
  await expect(page.locator(".admin-request-card").first()).toContainText(
    "Сильная протечка",
  );
  await expect(page.locator(".bottom-nav")).toHaveCount(0);
  await page.screenshot({
    path: "test-results/admin-queue-mobile.png",
    fullPage: true,
  });

  await page
    .getByRole("button", { name: /Просрочены.*срок уже вышел/ })
    .click();
  await expect(page.locator(".admin-request-card")).toHaveCount(1);
  await expect(page.locator(".admin-request-card")).toContainText(
    "Не работает лифт",
  );
  await expect(page).toHaveURL(/queue=overdue/);

  await page.getByLabel("Поиск по заявкам").fill("подъезд 2");
  await expect(page.locator(".admin-request-card")).toHaveCount(1);
  await expect(page).toHaveURL(/q=/);

  await page.getByRole("button", { name: "Сбросить фильтры" }).click();
  await expect(page).toHaveURL(/\/max\/admin$/);
  await expect(page.locator(".admin-request-card")).toHaveCount(2);

  for (const width of [320, 1280]) {
    await page.setViewportSize({ width, height: 900 });
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= window.innerWidth,
      ),
    ).toBe(true);
    if (width === 1280) {
      await page.screenshot({
        path: "test-results/admin-queue-desktop.png",
        fullPage: true,
      });
    }
  }
});

test("admin must describe the result before resolving a request", async ({
  page,
}) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  let current = { ...requests[1], status: "IN_PROGRESS" };
  const history = [
    {
      id: 1,
      requestId: current.id,
      status: "CREATED",
      createdAt: current.createdAt,
      comment: "",
      actor: "Житель",
    },
  ];
  let submittedBody: { status: string; comment: string } | null = null;

  await page.route("**/max/api/admin/requests/102", async (route) => {
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(current),
    });
  });
  await page.route("**/max/api/admin/requests/102/status", async (route) => {
    submittedBody = route.request().postDataJSON();
    current = { ...current, status: submittedBody!.status };
    history.push({
      id: 2,
      requestId: current.id,
      status: submittedBody!.status,
      createdAt: new Date().toISOString(),
      comment: submittedBody!.comment,
      actor: "УК · демо",
    });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(current),
    });
  });
  await page.route("**/max/api/admin/requests/102/history", (route) =>
    route.fulfill({ contentType: "application/json", body: JSON.stringify(history) }),
  );

  await page.goto("/max/admin/requests/102");

  await expect(page.getByLabel("Новый статус")).toHaveValue("RESOLVED");
  await expect(page.getByRole("button", { name: "Сохранить статус" })).toBeDisabled();
  await page.screenshot({
    path: "test-results/admin-detail-desktop.png",
    fullPage: true,
  });

  await page
    .getByLabel("Комментарий для жителя", { exact: false })
    .fill("  Лифт запущен, проверка выполнена.  ");
  await page.getByRole("button", { name: "Сохранить статус" }).click();

  await expect(page.locator(".current-status-row")).toContainText("Решено");
  expect(submittedBody).toEqual({
    status: "RESOLVED",
    comment: "Лифт запущен, проверка выполнена.",
  });
});
