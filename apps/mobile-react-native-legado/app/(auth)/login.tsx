import React, { useState } from "react";
import {
  KeyboardAvoidingView,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { useRouter } from "expo-router";
import { SafeAreaView } from "react-native-safe-area-context";
import { BrandLogo } from "@/components/ui/brand-logo";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuth } from "@/contexts/auth-context";
import { brandTokens } from "@/theme/tokens";

export default function LoginScreen() {
  const [enrollment, setEnrollment] = useState("");
  const [password, setPassword] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);

  const { login, error: authError, clearError } = useAuth();
  const router = useRouter();

  const isFormValid =
    enrollment.trim().length > 0 && password.trim().length > 0;

  const handleLogin = async () => {
    if (!isFormValid || isSubmitting) return;

    setIsSubmitting(true);
    clearError();

    try {
      await login(enrollment.trim(), password);
    } catch {
      // Error is set in AuthContext and rendered below
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <SafeAreaView style={styles.safeArea}>
      <KeyboardAvoidingView
        behavior={Platform.OS === "ios" ? "padding" : "height"}
        style={styles.keyboardAvoid}
      >
        <ScrollView
          contentContainerStyle={styles.scrollContent}
          keyboardShouldPersistTaps="handled"
        >
          <BrandLogo size={68} />

          <View style={styles.card}>
            <View style={styles.header}>
              <Text style={styles.title}>Entrar na sua conta</Text>
              <Text style={styles.subtitle}>
                Informe sua matrícula e senha para acessar
              </Text>
            </View>

            {authError ? (
              <View
                style={styles.errorContainer}
                accessibilityRole="alert"
                accessibilityLabel={authError}
              >
                <Text style={styles.errorText}>{authError}</Text>
              </View>
            ) : null}

            <Input
              label="Matrícula"
              placeholder="Digite sua matrícula"
              value={enrollment}
              onChangeText={(text) => {
                setEnrollment(text);
                if (authError) clearError();
              }}
              keyboardType="numeric"
              maxLength={10}
              returnKeyType="next"
              editable={!isSubmitting}
              accessibilityLabel="Matrícula"
              accessibilityHint="Digite o número da sua matrícula"
              testID="enrollment-input"
            />

            <Input
              label="Senha"
              placeholder="Digite sua senha"
              value={password}
              onChangeText={(text) => {
                setPassword(text);
                if (authError) clearError();
              }}
              secureTextEntry
              returnKeyType="done"
              onSubmitEditing={handleLogin}
              editable={!isSubmitting}
              accessibilityLabel="Senha"
              accessibilityHint="Digite sua senha de acesso"
              testID="password-input"
            />

            <Button
              title="Entrar"
              onPress={handleLogin}
              disabled={!isFormValid || isSubmitting}
              loading={isSubmitting}
              accessibilityLabel="Entrar"
              accessibilityHint="Toque para autenticar com matrícula e senha"
              style={styles.submitButton}
            />

            <View style={styles.linksContainer}>
              <Pressable
                onPress={() => router.push("/(auth)/ativar")}
                accessibilityRole="link"
                accessibilityLabel="Ativar conta"
                accessibilityHint="Navega para a tela de ativação de conta"
                style={styles.linkButton}
              >
                <Text style={styles.linkText}>Ativar conta</Text>
              </Pressable>

              <Text style={styles.linkDivider}>•</Text>

              <Pressable
                onPress={() => router.push("/(auth)/recuperar-senha")}
                accessibilityRole="link"
                accessibilityLabel="Esqueci minha senha"
                accessibilityHint="Navega para a tela de recuperação de senha"
                style={styles.linkButton}
              >
                <Text style={styles.linkText}>Esqueci minha senha</Text>
              </Pressable>
            </View>
          </View>

          <View style={styles.footer}>
            <Text style={styles.footerText}>
              Acesso seguro e protegido para atletas e treinadores
            </Text>
          </View>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: brandTokens.colors.surface950,
  },
  keyboardAvoid: {
    flex: 1,
  },
  scrollContent: {
    flexGrow: 1,
    justifyContent: "center",
    paddingHorizontal: brandTokens.spacing.md,
    paddingVertical: brandTokens.spacing.lg,
  },
  card: {
    backgroundColor: brandTokens.colors.surface900,
    borderRadius: brandTokens.borderRadius.lg,
    padding: brandTokens.spacing.lg,
    borderWidth: 1,
    borderColor: brandTokens.colors.borderDefault,
  },
  header: {
    marginBottom: brandTokens.spacing.lg,
  },
  title: {
    ...brandTokens.typography.title,
    color: brandTokens.colors.textPrimary,
    marginBottom: brandTokens.spacing.xs,
  },
  subtitle: {
    ...brandTokens.typography.body,
    color: brandTokens.colors.textSecondary,
  },
  errorContainer: {
    backgroundColor: brandTokens.colors.statusErrorBg,
    borderColor: brandTokens.colors.statusErrorBorder,
    borderWidth: 1,
    borderRadius: brandTokens.borderRadius.md,
    padding: brandTokens.spacing.md,
    marginBottom: brandTokens.spacing.md,
  },
  errorText: {
    ...brandTokens.typography.body,
    color: brandTokens.colors.textPrimary,
    fontWeight: "500",
  },
  submitButton: {
    marginTop: brandTokens.spacing.sm,
  },
  linksContainer: {
    flexDirection: "row",
    justifyContent: "center",
    alignItems: "center",
    marginTop: brandTokens.spacing.lg,
    gap: brandTokens.spacing.sm,
  },
  linkButton: {
    paddingVertical: brandTokens.spacing.xs,
    paddingHorizontal: brandTokens.spacing.xs,
  },
  linkText: {
    ...brandTokens.typography.caption,
    color: brandTokens.colors.brandGold300,
    fontWeight: "600",
  },
  linkDivider: {
    color: brandTokens.colors.textMuted,
  },
  footer: {
    marginTop: brandTokens.spacing.xl,
    alignItems: "center",
  },
  footerText: {
    ...brandTokens.typography.caption,
    color: brandTokens.colors.textMuted,
    textAlign: "center",
  },
});
