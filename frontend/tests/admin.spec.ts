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
  await page.setViewportSize({ width: 390, height: 844 });
  await page.route("**/max/api/admin/requests?*", (route) => {
    const url = new URL(route.request().url());
    const queue = url.searchParams.get("queue");
    let items = requests;
    if (queue === "ACTIVE") {
      items = items.filter((item) => !["CLOSED", "REJECTED"].includes(item.status));
    }
    if (queue === "DONE") {
      items = items.filter((item) => ["CLOSED", "REJECTED"].includes(item.status));
    }
    if (queue === "OVERDUE") {
      items = items.filter(
        (item) =>
          !["CLOSED", "REJECTED"].includes(item.status) &&
          new Date(item.deadline).getTime() < now,
      );
    }
    const query = (url.searchParams.get("q") ?? "").toLowerCase();
    if (query) items = items.filter((item) => `${item.id} ${item.address} ${item.description}`.toLowerCase().includes(query));
    return route.fulfill({ contentType: "application/json", body: JSON.stringify({ items, page: 1, pageSize: 20, total: items.length, summary: { active: 3, new: 1, overdue: 1, unassigned: 3, done: 1 } }) });
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
  await expect(page.locator(".admin-request-cards")).toContainText("Когда будет уборка двора");
  await expect(page.locator(".admin-stats").getByRole("button", { name: /Завершённые/ })).toContainText("1");
  await page.locator(".admin-stats").getByRole("button", { name: /Завершённые/ }).click();
  await expect(page.locator(".admin-request-card")).toHaveCount(1);
  await expect(page.locator(".admin-request-cards")).toContainText("Закрытая заявка на освещение");
  await expect(page.locator(".admin-request-cards")).not.toContainText("Когда будет уборка двора");
  await page.locator(".admin-stats").getByRole("button", { name: /Активные/ }).click();
  await expect(page.locator(".admin-request-card")).toHaveCount(3);
  await expect(page.locator(".bottom-nav")).toHaveCount(0);
  await page.screenshot({
    path: "test-results/admin-queue-mobile.png",
    fullPage: true,
  });

  await page
    .locator(".admin-stats")
    .getByRole("button", { name: /Просроченные/ })
    .click();
  await expect(page.locator(".admin-request-card")).toHaveCount(1);
  await expect(page.locator(".admin-request-card").filter({ hasText: "Не работает лифт" })).toHaveCount(1);
  await expect(page).toHaveURL(/queue=overdue/);

  await page.getByLabel("Поиск по заявкам").fill("подъезд 2");
  await expect(page.locator(".admin-request-card")).toHaveCount(1);
  await expect(page).toHaveURL(/q=/);

  await page.getByRole("button", { name: "Сбросить" }).click();
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

test("admin paginates requests on the server", async ({ page }) => {
  const many = Array.from({ length: 25 }, (_, index) => ({
    ...requests[0],
    id: String(index + 1),
    description: `Заявка ${index + 1}`,
  }));
  await page.route("**/max/api/admin/requests?*", (route) => {
    const url = new URL(route.request().url());
    const current = Number(url.searchParams.get("page") ?? "1");
    const size = Number(url.searchParams.get("pageSize") ?? "20");
    const items = many.slice((current - 1) * size, current * size);
    return route.fulfill({ contentType: "application/json", body: JSON.stringify({ items, page: current, pageSize: size, total: many.length, summary: { active: 25, new: 25, overdue: 0, unassigned: 25, done: 0 } }) });
  });
  await page.route("**/max/api/organizations", (route) => route.fulfill({ contentType: "application/json", body: "[]" }));
  await page.goto("/max/admin");
  await expect(page.locator(".admin-request-card")).toHaveCount(20);
  await page.getByRole("button", { name: "Далее" }).click();
  await expect(page).toHaveURL(/page=2/);
  await expect(page.locator(".admin-request-card")).toHaveCount(5);
  await expect(page.getByText("Показано 21–25 из 25")).toBeVisible();
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

  await page.route("**/max/api/admin/requests?*", (route) => {
    const queue = new URL(route.request().url()).searchParams.get("queue");
    const isDone = ["CLOSED", "REJECTED"].includes(current.status);
    const items = queue === "DONE"
      ? (isDone ? [current] : [])
      : (isDone ? [] : [current]);
    return route.fulfill({
      contentType: "application/json",
      body: JSON.stringify({
        items,
        page: 1,
        pageSize: 20,
        total: items.length,
        summary: {
          active: isDone ? 0 : 1,
          new: 0,
          overdue: isDone ? 0 : 1,
          unassigned: isDone ? 0 : 1,
          done: isDone ? 1 : 0,
        },
      }),
    });
  });
  await page.route("**/max/api/organizations", (route) =>
    route.fulfill({ contentType: "application/json", body: "[]" }),
  );
  await page.getByRole("link", { name: "← Заявки жителей" }).click();
  await expect(page.locator(".admin-request-card")).toContainText("Не работает лифт");
  await expect(page.locator(".admin-stats").getByRole("button", { name: /Завершённые/ })).toContainText("0");
  await page.locator(".admin-stats").getByRole("button", { name: /Завершённые/ }).click();
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
  await expect(page.locator(".admin-stats").getByRole("button", { name: /Завершённые/ })).toContainText("1");
  await expect(page.locator(".admin-request-card")).toHaveCount(0);
  await page.locator(".admin-stats").getByRole("button", { name: /Завершённые/ }).click();
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
  let rolePatch: unknown;
  let organizationPatch: unknown;

  await page.route("**/max/api/admin/requests?*", (route) =>
    route.fulfill({ contentType: "application/json", body: JSON.stringify({ items: [], page: 1, pageSize: 20, total: 0, summary: { active: 0, new: 0, overdue: 0, unassigned: 0, done: 0 } }) }),
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

  await page.goto("/max/admin/settings/users");

  await expect(
    page.getByRole("heading", { name: "Администрирование" }),
  ).toBeVisible();
  const managerRow = page.locator(".admin-user-row").filter({
    hasText: "manager@example.com",
  });
  await managerRow.getByLabel("Менеджер").click();
  await expect(managerRow.getByLabel("Менеджер")).toBeChecked();
  await managerRow.getByRole("combobox", { name: "Организация" }).selectOption("1");

  expect(rolePatch).toEqual({ roles: ["resident", "manager"] });
  await expect.poll(() => organizationPatch).toEqual({ organizationId: "1" });
});
