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
    deadline: hoursFromNow(48),
    createdAt: hoursFromNow(-96),
    address: "г. Москва, ул. Тестовая, д. 1",
    kind: "QUESTION",
    text: "Закрытый вопрос про уборку.",
    hasPhoto: false,
  },
  {
    id: "104",
    userId: "1",
    houseId: "1",
    description: "Закрытая заявка на освещение",
    problemType: "OTHER",
    responsibleOrganizationId: "1",
    responsibleOrganization: "УК «Тестовая»",
    status: "CLOSED",
    deadline: hoursFromNow(-72),
    createdAt: hoursFromNow(-120),
    address: "г. Москва, ул. Тестовая, д. 1",
    kind: "PROBLEM",
    text: "Освещение восстановлено.",
    hasPhoto: false,
  },
];

test.beforeEach(async ({ page }) => {
  await page.route("https://st.max.ru/js/max-web-app.js", (route) =>
    route.abort(),
  );
  await page.route("**/max/api/me", (route) =>
    route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        id: "1",
        email: "admin@example.com",
        name: "admin@example.com",
        roles: ["admin", "resident"],
      }),
    }),
  );
  await page.route("**/max/api/admin/users", (route) =>
    route.fulfill({ contentType: "application/json", body: "[]" }),
  );
});

test("admin queue prioritizes work and keeps filters in the URL", async ({
  page,
}) => {
  await page.route(/\/max\/api\/admin\/requests\?/, (route) => {
    const queue = new URL(route.request().url()).searchParams.get("queue");
    const result = queue === "ALL"
      ? requests
      : requests.filter(({ status }) => status !== "CLOSED" && status !== "REJECTED");
    return route.fulfill({ contentType: "application/json", body: JSON.stringify(result) });
  });
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

  await expect(page.locator(".admin-request-card")).toHaveCount(3);
  await expect(page.locator(".admin-request-card").first()).toContainText(
    "Сильная протечка",
  );
  await expect(page.locator(".admin-request-list")).toContainText("Когда будет уборка двора");
  await expect(page.getByRole("button", { name: /Завершены.*решены или закрыты/ })).toContainText("1");
  await page.getByRole("button", { name: "Завершённые" }).click();
  await expect(page.locator(".admin-request-card")).toHaveCount(1);
  await expect(page.locator(".admin-request-list")).toContainText("Закрытая заявка на освещение");
  await expect(page.locator(".admin-request-list")).not.toContainText("Когда будет уборка двора");
  await page.getByRole("button", { name: "Активные", exact: true }).click();
  await expect(page.locator(".admin-request-card")).toHaveCount(3);
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
  await expect(page.locator(".admin-request-card")).toHaveCount(3);

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

  await expect(page.locator(".current-status-row")).toContainText("Ожидает подтверждения");
  await expect(page.getByText("Работы выполнены. Ожидается подтверждение жителя.")).toBeVisible();
  expect(submittedBody).toEqual({
    status: "RESOLVED",
    comment: "Лифт запущен, проверка выполнена.",
  });

  await page.route(/\/max\/api\/admin\/requests\?/, (route) =>
    route.fulfill({ contentType: "application/json", body: JSON.stringify([current]) }),
  );
  await page.route("**/max/api/organizations", (route) =>
    route.fulfill({ contentType: "application/json", body: "[]" }),
  );
  await page.getByRole("link", { name: "← Заявки жителей" }).click();
  await expect(page.locator(".admin-request-card")).toContainText("Не работает лифт");
  await expect(page.getByRole("button", { name: /Завершены.*решены или закрыты/ })).toContainText("0");
  await page.getByRole("button", { name: "Завершённые" }).click();
  await expect(page.locator(".admin-request-card")).toHaveCount(0);

  await page.route("**/max/api/requests/102", (route) =>
    route.fulfill({ contentType: "application/json", body: JSON.stringify(current) }),
  );
  await page.route("**/max/api/requests/102/history", (route) =>
    route.fulfill({ contentType: "application/json", body: JSON.stringify(history) }),
  );
  await page.route("**/max/api/requests/102/messages", (route) =>
    route.fulfill({ contentType: "application/json", body: "[]" }),
  );
  await page.route("**/max/api/requests/102/resolution", (route) => {
    expect(route.request().postDataJSON()).toMatchObject({ solved: true });
    current = { ...current, status: "CLOSED" };
    return route.fulfill({ contentType: "application/json", body: JSON.stringify(current) });
  });
  await page.goto("/max/requests/102");
  await page.getByRole("button", { name: "Подтвердить решение" }).click();
  await expect(page.getByText("Закрыто")).toBeVisible();

  await page.goto("/max/admin");
  await expect(page.getByRole("button", { name: /Завершены.*решены или закрыты/ })).toContainText("1");
  await expect(page.locator(".admin-request-card")).toHaveCount(0);
  await page.getByRole("button", { name: "Завершённые" }).click();
  await expect(page.locator(".admin-request-card")).toContainText("Не работает лифт");
});

test("admin manages organizations and user access", async ({ page }) => {
  const organizations = [{ id: "1", name: "УК «Тестовая»" }];
  const users = [
    {
      id: "1",
      email: "admin@example.com",
      name: "admin@example.com",
      roles: ["admin", "resident"],
    },
    {
      id: "2",
      email: "manager@example.com",
      name: "manager@example.com",
      roles: ["resident"],
    },
  ];
  const emptyRequests: unknown[] = [];
  let rolePatch: unknown;
  let organizationPatch: unknown;

  await page.route(/\/max\/api\/admin\/requests\?/, (route) =>
    route.fulfill({ contentType: "application/json", body: JSON.stringify(emptyRequests) }),
  );
  await page.route("**/max/api/organizations", (route) =>
    route.fulfill({ contentType: "application/json", body: JSON.stringify(organizations) }),
  );
  await page.route("**/max/api/admin/users**", (route) => {
    const url = new URL(route.request().url());
    if (route.request().method() === "GET" && url.pathname.endsWith("/admin/users")) {
      return route.fulfill({
        contentType: "application/json",
        body: JSON.stringify(users),
      });
    }
    return route.fallback();
  });
  await page.route("**/max/api/admin/organizations", async (route) => {
    const body = route.request().postDataJSON();
    organizations.push({ id: "2", name: body.name });
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(organizations[1]),
    });
  });
  await page.route("**/max/api/admin/users/2/roles", async (route) => {
    rolePatch = route.request().postDataJSON();
    users[1].roles = (rolePatch as { roles: string[] }).roles;
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(users[1]),
    });
  });
  await page.route("**/max/api/admin/users/2/organization", async (route) => {
    organizationPatch = route.request().postDataJSON();
    users[1].organizationId = (organizationPatch as { organizationId: string }).organizationId;
    await route.fulfill({
      contentType: "application/json",
      body: JSON.stringify(users[1]),
    });
  });

  await page.goto("/max/admin");

  await expect(
    page.getByRole("heading", { name: "Администрирование" }),
  ).toBeVisible();
  await page.getByLabel("Название УК").fill("ООО УК Новый дом");
  await page.getByRole("button", { name: "Создать УК" }).click();
  await expect(
    page.locator(".admin-filters select").first(),
  ).toContainText("ООО УК Новый дом");

  const managerRow = page.locator(".admin-user-row").filter({
    hasText: "manager@example.com",
  });
  await managerRow.getByLabel("Менеджер").click();
  await expect(managerRow.getByLabel("Менеджер")).toBeChecked();
  await managerRow.getByRole("combobox", { name: "Организация" }).selectOption("2");

  expect(rolePatch).toEqual({ roles: ["resident", "manager"] });
  expect(organizationPatch).toEqual({ organizationId: "2" });
});
