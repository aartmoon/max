import { expect, test, type Page } from "@playwright/test";

const baseRequest = {
  id: "77", userId: "1", houseId: "4", description: "Не работает отопление", problemType: "HEATING",
  responsibleOrganizationId: "1004", responsibleOrganization: "ПАО «МОЭК»", primaryOrganizationId: "1004", primaryOrganization: "ПАО «МОЭК»",
  contractorOrganizationId: "", contractorOrganization: "", status: "RESOLVED", deadline: "2026-10-01T10:00:00Z", createdAt: "2026-09-29T10:00:00Z",
  address: "г. Москва, ул. Арбат, д. 24", houseObjectId: "67036019", kind: "PROBLEM", text: "Заявка", hasPhoto: false,
  problemPlace: "RESOURCE_INPUT", urgency: "NORMAL", routingReason: "Правило дома: категория «HEATING». Демонстрационное правило", routingSource: "ГИС ЖКХ",
  assignedUserId: "2", assignedUserName: "Иван Петров", visitStart: "2026-09-30T09:00:00Z", visitEnd: "2026-09-30T11:00:00Z", executorContact: "+7 900 000-00-00",
  finalReport: "Подача тепла восстановлена", awaitingParty: "NONE", reopenCount: 0,
};

async function common(page: Page, user: object) {
  await page.route("https://st.max.ru/js/max-web-app.js", (route) => route.abort());
  await page.route("**/max/api/me", (route) => route.fulfill({ contentType: "application/json", body: JSON.stringify(user) }));
}

test("resident cannot open organization console and can close resolved request", async ({ page }) => {
  await common(page, { id: "1", email: "resident@example.test", name: "Житель", roles: ["resident"] });
  let current = { ...baseRequest };
  await page.route("**/max/api/requests/77", (route) => route.fulfill({ contentType: "application/json", body: JSON.stringify(current) }));
  await page.route("**/max/api/requests/77/history", (route) => route.fulfill({ contentType: "application/json", body: "[]" }));
  await page.route("**/max/api/requests/77/messages", (route) => route.fulfill({ contentType: "application/json", body: "[]" }));
  await page.route("**/max/api/requests/77/resolution", async (route) => { expect(route.request().postDataJSON()).toEqual({ solved: true, rating: 5, comment: "Работы приняты" }); current = { ...current, status: "CLOSED" }; await route.fulfill({ contentType: "application/json", body: JSON.stringify(current) }); });
  await page.goto("/max/admin"); await expect(page).toHaveURL(/\/max\/?$/);
  await page.goto("/max/requests/77");
  await expect(page.getByText("ПАО «МОЭК»", { exact: true })).toBeVisible();
  await expect(page.getByText("Подача тепла восстановлена")).toBeVisible();
  await page.getByLabel("Комментарий").fill("Работы приняты");
  await page.getByRole("button", { name: "Подтвердить решение" }).click();
  await expect(page.getByText("Закрыто")).toBeVisible();
});

test("organization sees internal notes and message retry preserves text", async ({ page }) => {
  await common(page, { id: "2", email: "manager@example.test", name: "Менеджер", roles: ["manager"], organizationId: "1004" });
  const request = { ...baseRequest, status: "IN_PROGRESS", finalReport: "" };
  await page.route("**/max/api/admin/requests/77", (route) => route.fulfill({ contentType: "application/json", body: JSON.stringify(request) }));
  await page.route("**/max/api/admin/requests/77/history", (route) => route.fulfill({ contentType: "application/json", body: "[]" }));
  await page.route("**/max/api/organization/staff*", (route) => route.fulfill({ contentType: "application/json", body: JSON.stringify([{ id: "2", name: "Менеджер", email: "manager@example.test", roles: ["manager"], organizationId: "1004" }]) }));
  await page.route("**/max/api/organization/requests/77/messages", (route) => {
    if (route.request().method() === "POST") return route.fulfill({ status: 503, contentType: "application/json", body: JSON.stringify({ error: "Не удалось отправить" }) });
    return route.fulfill({ contentType: "application/json", body: JSON.stringify([{ id: "9", requestId: "77", type: "INTERNAL_NOTE", text: "Позвонить исполнителю", authorName: "Менеджер", authorRole: "manager", createdAt: "2026-09-29T12:00:00Z", attachments: [] }]) });
  });
  await page.goto("/max/admin/requests/77");
  await expect(page.getByText("Позвонить исполнителю")).toBeVisible();
  await page.getByLabel("Новое сообщение").fill("Ответ жителю");
  await page.getByRole("button", { name: "Отправить" }).click();
  await expect(page.getByRole("alert")).toContainText("Не удалось отправить");
  await expect(page.getByLabel("Новое сообщение")).toHaveValue("Ответ жителю");
});

test("resident API error does not expose a foreign request", async ({ page }) => {
  await common(page, { id: "1", email: "resident@example.test", name: "Житель", roles: ["resident"] });
  await page.route("**/max/api/requests/999", (route) => route.fulfill({ status: 403, contentType: "application/json", body: JSON.stringify({ error: "Недостаточно прав" }) }));
  await page.route("**/max/api/requests/999/history", (route) => route.fulfill({ status: 403, contentType: "application/json", body: JSON.stringify({ error: "Недостаточно прав" }) }));
  await page.goto("/max/requests/999");
  await expect(page.getByRole("alert")).toContainText("Недостаточно прав");
});
