import React from "react";

export const View = ({ children, style, ...props }: any) =>
  React.createElement("div", { style, ...props }, children);

export const Text = ({ children, style, ...props }: any) =>
  React.createElement("span", { style, ...props }, children);

export const TextInput = ({
  onChangeText,
  value,
  placeholder,
  secureTextEntry,
  editable,
  ...props
}: any) =>
  React.createElement("input", {
    value,
    placeholder,
    disabled: editable === false,
    type: secureTextEntry ? "password" : "text",
    onChange: (e: any) => onChangeText?.(e.target.value),
    ...props,
  });

export const Pressable = ({
  children,
  onPress,
  disabled,
  style,
  ...props
}: any) =>
  React.createElement(
    "button",
    {
      onClick: disabled ? undefined : onPress,
      disabled,
      style: typeof style === "function" ? style({ pressed: false }) : style,
      ...props,
    },
    children,
  );

export const ActivityIndicator = ({ ...props }: any) =>
  React.createElement("span", { "data-testid": "activity-indicator", ...props });

export const Image = ({ source, ...props }: any) =>
  React.createElement("img", { src: source, ...props });

export const ScrollView = ({ children, ...props }: any) =>
  React.createElement("div", { ...props }, children);

export const KeyboardAvoidingView = ({ children, ...props }: any) =>
  React.createElement("div", { ...props }, children);

export const SafeAreaView = ({ children, style, ...props }: any) =>
  React.createElement("div", { style, ...props }, children);

export const SafeAreaProvider = ({ children, ...props }: any) =>
  React.createElement("div", { ...props }, children);

export const useSafeAreaInsets = () => ({ top: 0, bottom: 0, left: 0, right: 0 });

export const StyleSheet = {
  create: (styles: any) => styles,
};

export const Platform = {
  OS: "android",
  select: (obj: any) => obj.android || obj.default,
};
